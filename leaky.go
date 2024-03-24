package ratelimite

import (
	"fmt"
	"time"
)

// 漏桶限流器
// leaky限流无法应对突发流量，并发性比较弱，请求之间的时间间隔要大于PerRequest (maxPerMinute/time.Minute)
// ref: https://github.com/uber-go/ratelimit && https://segmentfault.com/a/1190000039304299
type LeakyLimiter struct {
	base
	// 每个请求的时间单元，即漏桶的速率
	PerRequest time.Duration
	// 松弛时间单元
	Slack time.Duration
	// 休眠时间
	RestTime time.Duration
	Lastest  time.Time
}

func newLeaky(bl *base) *LeakyLimiter {
	if bl.slack <= 1 {
		bl.slack = 5
	}

	perRequest := time.Minute / time.Duration(bl.maxPerMinute)
	l := &LeakyLimiter{
		base:       *bl,
		PerRequest: perRequest,
		Slack:      -1 * time.Duration(bl.slack) * perRequest,
	}
	l.RestTime = l.Slack

	return l
}

func (l *LeakyLimiter) Acquired() (time.Duration, bool, error) {
	now := l.clock.Now()
	// 第一次请求
	if l.Lastest.IsZero() {
		l.Lastest = now
		return 0, true, nil
	}

	lc := 0
trylock: // 尝试3次获取锁，每次间隔1s
	if err := l.mux.Lock(); err != nil {
		if lc < 3 {
			lc++
			time.Sleep(time.Second)
			goto trylock
		}
		// l.mux.Unlock()
		return 0, false, fmt.Errorf("Lock Fail")
	}
	defer l.mux.Unlock()

	if l.mux.getType() == REDIS {
		if err := FromRedisBytes(l); err != nil {
			return 0, false, fmt.Errorf("FromRedisBytes error")
		}
		defer ToRedisBytes(l)
	}
	// 累加，以便处理时间少的请求，缓冲处理时间比较长的请求
	l.RestTime += l.PerRequest - now.Sub(l.Lastest)
	// 请求的处理时间过长时，设置最大松弛时间
	if l.RestTime < l.Slack {
		l.RestTime = l.Slack
	}

	if l.RestTime > 0 {
		waitTime := l.RestTime
		l.RestTime = 0
		return waitTime, false, nil
	}
	l.Lastest = now

	return 0, true, nil
}
