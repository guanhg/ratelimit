package ratelimite

import (
	"time"
)

var _ Limiter = (*LeakyLimiter)(nil)

// 漏桶限流器
// leaky限流无法应对突发流量，并发性比较弱，请求之间的时间间隔要大于PerRequest (maxPerMinute/time.Minute)
// ref: https://github.com/uber-go/ratelimit && https://segmentfault.com/a/1190000039304299
type LeakyLimiter struct {
	*base

	RestTime int64 // 休眠时间
	Lastest  int64 // 上一次请求时间
}

func NewLeakyLimiter(threshold int, opts ...Option) *LeakyLimiter {
	l := &LeakyLimiter{
		base: newBase(threshold, opts...),
	}
	l.RestTime = l.slackTime()

	return l
}

func (l *LeakyLimiter) Type() Type {
	return Leaky
}

func (l *LeakyLimiter) Wait() {
	if d, b := l.Take(); !b {
		currClock.Wait(d)
	}
}

func (l *LeakyLimiter) WaitMax(max time.Duration) {
	if d, b := l.Take(); !b {
		currClock.WaitMax(d, max)
	}
}

// 松弛时间，即在请求紧张时，允许slack个请求的弹性
func (l *LeakyLimiter) slackTime() int64 {
	return int64(-l.slack * int(l.PerRate()))
}

func (l *LeakyLimiter) Take() (time.Duration, bool) {
	now := currClock.CurrentTimeMillis()
	// 第一次请求
	if l.Lastest == 0 {
		l.Lastest = now
		return 0, true
	}

	l.mux.Lock()
	defer l.mux.Unlock()

	// leaky logic
	// 累加，以便处理时间少的请求，缓冲处理时间比较长的请求
	l.RestTime += int64(l.PerRate()) + l.Lastest - now
	// 请求的处理时间过长时，设置最大松弛时间
	if l.RestTime < l.slackTime() {
		l.RestTime = l.slackTime()
	}

	if l.RestTime > 0 {
		waitTime := l.RestTime
		l.RestTime = 0
		return time.Duration(waitTime * int64(time.Millisecond)), false
	}
	l.Lastest = now

	return 0, true
}
