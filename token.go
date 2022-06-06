package ratelimite

import (
	"time"
)

// 令牌桶限流
// ref https://github.com/juju/ratelimit/blob/f60b32039441cd828005f82f3a54aafd00bc9882/ratelimit.go#L305
type TokenLimiter struct {
	baseLimiter
	// 可用token数
	Available int64
	// 最近填充token的时间
	Lastest time.Time
}

func newToken(bl *baseLimiter, ops ...Option) *TokenLimiter {
	for _, opt := range ops {
		opt(bl)
	}

	l := &TokenLimiter{
		baseLimiter: *bl,
		Available:   bl.maxReq,
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

func (t *TokenLimiter) Release() (time.Duration, bool, error) {
	return t.ReleaseN(1)
}

func (t *TokenLimiter) ReleaseN(n int64) (time.Duration, bool, error) {
	t.ato.Lock()
	defer t.ato.Unlock()

	if err := t.restore(); err != nil {
		return FOREVER, false, err
	}
	defer t.store()

	now := t.clock.Now()
	t.adjustAvailabe(now)
	t.Available -= n
	if t.Available >= 0 {
		return 0, true, nil
	}

	waitTime := time.Duration(1-t.Available) * t.unit
	return waitTime, false, nil
}

func (t *TokenLimiter) adjustAvailabe(now time.Time) {
	// 第一次请求
	if t.Lastest.IsZero() {
		t.Lastest = now
		return
	}
	incr := int64(now.Sub(t.Lastest) / t.unit)
	if t.Available >= t.maxReq {
		return
	}

	t.Available += incr
	if t.Available > t.maxReq {
		t.Available = t.maxReq
	}

	t.Lastest = now
}

func (t *TokenLimiter) Reset() error {
	t.ato.Lock()
	defer t.ato.Unlock()

	if err := t.restore(); err != nil {
		return err
	}
	defer t.store()

	t.Available = t.maxReq
	t.Lastest = time.Time{}
	return nil
}

func (t *TokenLimiter) restore() error {
	data, err := t.ato.Restore()
	if err != nil || len(data) == 0 {
		return err
	}
	return t.deserializeFunc(data, t)
}

func (t *TokenLimiter) store() error {
	data, err := t.serializeFunc(t)
	if err != nil {
		return err
	}
	return t.ato.Store(data)
}
