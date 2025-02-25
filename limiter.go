package ratelimite

import (
	"sync"
	"time"
)

type Type string

const (
	Count Type = "Count"
	Leaky Type = "Leaky"
	Token Type = "Token"
)

type Limiter interface {
	Take() (time.Duration, bool)
	Wait()
	WaitMax(maxDuration time.Duration)
	Type() Type
}

func init() {
	currClock = NewRealClock()
}

type base struct {
	mux sync.Mutex

	threshold     int   // 阈值
	thresholdInMs int64 // 阈值的时间跨度，单位毫秒，默认1000
	bucketNum     int   // 用于计数限流器，bucket数量，默认10
	slack         int   `json:"slack"` // 用于漏桶限流，默认5
}

func newBase(threshold int, opts ...Option) *base {
	bl := &base{
		threshold:     threshold,
		thresholdInMs: 1000,
		bucketNum:     10,
		slack:         5,
	}
	for _, opt := range opts {
		opt(bl)
	}

	return bl
}

// 每个请求的平均时间(单位毫秒)
func (b *base) PerRate() float64 {
	return float64(b.thresholdInMs) / float64(b.threshold)
}

type Option func(*base)

func WithThresholdInMs(ms int64) Option {
	return func(bl *base) {
		bl.thresholdInMs = ms
	}
}

// 用于计数限流器, 为了平滑, 把限流器的单位时间分成ins间隔
func WithBucketlNum(num int) Option {
	return func(bl *base) {
		bl.bucketNum = num
	}
}

// 用于漏桶限流器, 为了平滑, 增加松弛因子
func WithSlack(slack int) Option {
	return func(bl *base) {
		bl.slack = slack
	}
}

type Clock interface {
	Now() time.Time
	CurrentTimeMillis() uint64
	CurrentTimeNano() uint64
	Wait(time.Duration)
	MaxWait(d time.Duration, md time.Duration)
}

var currClock *realClock

type realClock struct{}

func NewRealClock() *realClock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (t *realClock) CurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func (t *realClock) CurrentTimeNano() int64 {
	return t.Now().UnixNano()
}

func (c *realClock) Wait(d time.Duration) {
	time.Sleep(d)
}

func (c *realClock) WaitMax(d time.Duration, md time.Duration) {
	if d > md {
		c.Wait(md)
	} else {
		c.Wait(d)
	}
}
