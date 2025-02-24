package ratelimite

import (
	"time"
)

var _ Limiter = (*TokenLimiter)(nil)

// 令牌桶限流
// ref https://github.com/juju/ratelimit/blob/f60b32039441cd828005f82f3a54aafd00bc9882/ratelimit.go#L305
type TokenLimiter struct {
	*base

	Available int64 // 可用token数
	Lastest   int64 // 最近填充token的时间
}

func NewTokenLimiter(threshold int, opts ...Option) *TokenLimiter {
	t := &TokenLimiter{
		base: newBase(threshold, opts...),
	}
	t.Available = int64(t.threshold)

	return t
}

func (t *TokenLimiter) Type() Type {
	return Token
}

func (t *TokenLimiter) Wait() {
	if d, b := t.Take(); !b {
		currClock.Wait(d)
	}
}

func (t *TokenLimiter) WaitMax(max time.Duration) {
	if d, b := t.Take(); !b {
		currClock.WaitMax(d, max)
	}
}

func (t *TokenLimiter) Take() (time.Duration, bool) {
	return t.TakeN(1)
}

func (t *TokenLimiter) TakeN(n int64) (time.Duration, bool) {
	t.mux.Lock()
	defer t.mux.Unlock()

	now := currClock.CurrentTimeMillis()
	t.adjustAvailabe(now)
	t.Available -= n
	if t.Available >= 0 {
		return 0, true
	}

	// 等待时间=令牌数*单个令牌时间，注意，这里的令牌数位负值
	waitMs := (1 - t.Available) * int64(t.PerRate())
	return time.Duration(waitMs * int64(time.Millisecond)), false
}

func (t *TokenLimiter) adjustAvailabe(now int64) {
	// 第一次请求
	if t.Lastest == 0 {
		t.Lastest = now
		return
	}
	incr := int64(float64(now-t.Lastest) / t.PerRate())
	if t.Available >= int64(t.threshold) {
		return
	}

	t.Available += incr
	if t.Available > int64(t.threshold) {
		t.Available = int64(t.threshold)
	}

	t.Lastest = now
}
