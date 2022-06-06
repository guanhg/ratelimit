package ratelimite

import (
	"time"
)

// 漏桶限流器
// ref: https://github.com/uber-go/ratelimit && https://segmentfault.com/a/1190000039304299
type LeakyLimiter struct {
	baseLimiter
	// 请求时间单元，松弛时间单元
	PerRequest time.Duration
	PerSlack   time.Duration
	// 休眠时间
	RestTime time.Duration
	Lastest  time.Time
}

func newLeaky(bl *baseLimiter, ops ...Option) *LeakyLimiter {
	for _, opt := range ops {
		opt(bl)
	}
	if bl.slack <= 0 {
		bl.slack = 1
	}

	perRequest := bl.unit / time.Duration(bl.maxReq)
	l := &LeakyLimiter{
		baseLimiter: *bl,
		PerRequest:  perRequest,
		PerSlack:    -1 * time.Duration(bl.slack) * perRequest,
	}

	ato, err := l.atomicFunc(l.storageType, l.storageAddr)
	if err != nil {
		panic(err)
	}
	if l.storageType == REDIS_TYPE {
		ato.(*RedisAtomic).SetKey(l.name)
	}
	l.ato = ato

	return l
}

func (l *LeakyLimiter) Release() (time.Duration, bool, error) {
	l.ato.Lock()
	defer l.ato.Unlock()

	if err := l.restore(); err != nil {
		return FOREVER, false, err
	}
	defer l.store()

	now := l.clock.Now()
	// 第一次请求
	if l.Lastest.IsZero() {
		l.Lastest = now
		return 0, true, nil
	}
	// 累加，以便处理时间少的请求，缓冲处理时间比较长的请求
	l.RestTime += l.PerRequest - now.Sub(l.Lastest)
	// 请求的处理时间过长时，设置最大松弛时间
	if l.RestTime < l.PerSlack {
		l.RestTime = l.PerSlack
	}

	if l.RestTime > 0 {
		waitTime := l.RestTime
		l.RestTime = 0
		return waitTime, false, nil
	}
	l.Lastest = now

	return 0, true, nil
}

func (l *LeakyLimiter) Reset() error {
	l.ato.Lock()
	defer l.ato.Unlock()

	if err := l.restore(); err != nil {
		return err
	}
	defer l.store()

	l.RestTime = 0
	l.Lastest = time.Time{}

	return nil
}

func (l *LeakyLimiter) restore() error {
	data, err := l.ato.Restore()
	if err != nil || len(data) == 0 {
		return err
	}
	return l.deserializeFunc(data, l)
}

func (l *LeakyLimiter) store() error {
	data, err := l.serializeFunc(l)
	if err != nil {
		return err
	}
	return l.ato.Store(data)
}
