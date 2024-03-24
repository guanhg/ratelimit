package ratelimite

import (
	"encoding/json"
	"time"
)

type LimiterType string

const (
	Counter LimiterType = "Count"
	Leaky   LimiterType = "Leaky"
	Token   LimiterType = "Token"
)

type Limiter interface {
	Acquired() (time.Duration, bool, error)
	SetRedis(string) error
	getStorage() storage
}

type base struct {
	name  string      `json:"-"`
	clock Clock       `json:"-"`
	_type LimiterType `json:"-"`
	mux   Atomic      `json:"-"`

	maxPerMinute int64 // 每分钟最大请求数
	// smooth option
	ins   int `json:"-"` // used to smoothing in CounterLimiter
	slack int `json:"-"` // used to smoothing in LeakyLimiter
}

func NewLimiter(name string, t LimiterType, maxPerMinute int64, opts ...Option) Limiter {
	bl := &base{
		name:         name,
		clock:        realClock{},
		_type:        t,
		maxPerMinute: maxPerMinute,
		mux:          NewAtomic(PROCESS),
	}
	for _, opt := range opts {
		opt(bl)
	}

	switch t {
	case Leaky:
		return newLeaky(bl)
	case Token:
		return newToken(bl)
	}

	return nil
}

func (bl *base) SetRedis(redisAddr string) error {
	mux := NewAtomic(REDIS).(*RedisAtomic)
	if err := mux.SetAddr(redisAddr); err != nil {
		return err
	}
	mux.SetKey(bl.name)
	bl.mux = mux
	return nil
}

func (bl *base) SetProcess() {
	bl.mux = NewAtomic(PROCESS).(*ProcessAtomic)
}

func (bl *base) getStorage() storage {
	return bl.mux.(storage)
}

func (bl *base) Wait(d time.Duration) {
	bl.clock.Wait(d)
}

func (bl *base) WaitMax(d time.Duration, md time.Duration) {
	bl.clock.MaxWait(d, md)
}

type Option func(*base)

// 用于计数限流器, 为了平滑, 把限流器的单位时间分成ins间隔
func WithIns(ins int) Option {
	return func(bl *base) {
		bl.ins = ins
	}
}

// 用于漏桶限流器, 为了平滑, 增加松弛因子
func WithSlack(slack int) Option {
	return func(bl *base) {
		bl.slack = slack
	}
}

func ToRedisBytes(l Limiter) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return l.getStorage().Store(data)
}

func FromRedisBytes(l Limiter) error {
	data, err := l.getStorage().Restore()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, l)
}

type Clock interface {
	Now() time.Time
	Wait(time.Duration)
	MaxWait(d time.Duration, md time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Wait(d time.Duration) {
	time.Sleep(d)
}

func (c realClock) MaxWait(d time.Duration, md time.Duration) {
	if d > md {
		c.Wait(md)
	} else {
		c.Wait(d)
	}
}
