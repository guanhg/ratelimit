package ratelimite

import (
	"fmt"
	"time"
)

// 令牌桶限流
// ref https://github.com/juju/ratelimit/blob/f60b32039441cd828005f82f3a54aafd00bc9882/ratelimit.go#L305
type TokenLimiter struct {
	base
	// 可用token数
	Available int64
	// 最近填充token的时间
	Lastest time.Time
}

func newToken(bl *base) *TokenLimiter {
	l := &TokenLimiter{
		base:      *bl,
		Available: bl.maxPerMinute,
	}

	return l
}

func (t *TokenLimiter) Acquired() (time.Duration, bool, error) {
	return t.AcquiredN(1)
}

func (t *TokenLimiter) AcquiredN(n int64) (time.Duration, bool, error) {
	lc := 0
trylock: // 尝试3次获取锁，每次间隔1s
	if err := t.mux.Lock(); err != nil {
		if lc < 3 {
			lc++
			time.Sleep(time.Second)
			goto trylock
		}
		// t.mux.Unlock()
		return 0, false, fmt.Errorf("Lock Fail")
	}
	defer t.mux.Unlock()

	if t.mux.getType() == REDIS {
		if err := FromRedisBytes(t); err != nil {
			return 0, false, fmt.Errorf("FromRedisBytes error")
		}
		defer ToRedisBytes(t)
	}

	now := t.clock.Now()
	t.adjustAvailabe(now)
	t.Available -= n
	if t.Available >= 0 {
		return 0, true, nil
	}

	waitTime := time.Duration(float64(1-t.Available) * float64(time.Minute) / float64(t.maxPerMinute))
	return waitTime, false, nil
}

func (t *TokenLimiter) adjustAvailabe(now time.Time) {
	// 第一次请求
	if t.Lastest.IsZero() {
		t.Lastest = now
		return
	}
	incr := int64(time.Duration(t.maxPerMinute) * now.Sub(t.Lastest) / time.Minute)
	if t.Available >= t.maxPerMinute {
		return
	}

	t.Available += incr
	if t.Available > t.maxPerMinute {
		t.Available = t.maxPerMinute
	}

	t.Lastest = now
}
