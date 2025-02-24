package ratelimite

import (
	"time"
)

var _ Limiter = (*CountLimiter)(nil)

type CountLimiter struct {
	*base
	buckets *SlideBucket
}

func NewCountLimiter(threshold int, opts ...Option) *CountLimiter {
	c := &CountLimiter{
		base: newBase(threshold, opts...),
	}
	bucketInMs := c.bucketInMs()
	now := currClock.CurrentTimeMillis()

	c.buckets = NewSolideBucket(c.bucketNum, bucketInMs, now)
	return c
}

func (c *CountLimiter) Type() Type {
	return Count
}

func (c *CountLimiter) Wait() {
	if d, b := c.Take(); !b {
		currClock.Wait(d)
	}
}

func (c *CountLimiter) WaitMax(max time.Duration) {
	if d, b := c.Take(); !b {
		currClock.WaitMax(d, max)
	}
}

func (c *CountLimiter) Take() (time.Duration, bool) {
	return c.TakeN(1)
}

func (c *CountLimiter) TakeN(n int) (time.Duration, bool) {
	// count logic
	now := currClock.CurrentTimeMillis()
	c.buckets.AddWithTime(n, now)
	count := c.buckets.Count(now)
	if count > c.threshold {
		return time.Duration(c.bucketInMs()) * time.Millisecond, false
	}
	return 0, true
}

// 一个bucket的时间跨度
func (c *CountLimiter) bucketInMs() int64 {
	return c.thresholdInMs / int64(c.bucketNum)
}
