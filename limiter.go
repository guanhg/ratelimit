package ratelimite

import (
	"encoding/json"
	"time"
)

// Limiter error
type LimiterError struct {
	text string
}

func (e *LimiterError) Error() string {
	return e.text
}

type StorageType uint8

const (
	UNKNOW_TYPE  StorageType = 0
	PROCESS_TYPE StorageType = 1 // memroy storage
	REDIS_TYPE   StorageType = 2
)

type LimiterType string

const (
	Counter LimiterType = "Counter"
	Leaky   LimiterType = "Leaky"
	Token   LimiterType = "Token"
)
const FOREVER time.Duration = 0x7fffffffffffffff

type Limiter interface {
	Reset() error
	Release() (time.Duration, bool, error)
	// Update(p Parameter) error
}

type (
	AtomicFunc      func(StorageType, string) (Atomic, error)
	DeserializeFunc func([]byte, Limiter) error
	SerializeFunc   func(Limiter) ([]byte, error)
)

type baseLimiter struct {
	clock       Clock       `json:"-"`
	name        string      `json:"-"`
	storageType StorageType `json:"-"`
	storageAddr string      `json:"-"`
	limiterType LimiterType `json:"-"`
	unit        time.Duration
	maxReq      int64
	ato         Atomic `json:"-"`
	// smooth option
	ins   int `json:"-"` // used to smoothing in CounterLimiter
	slack int `json:"-"` // used to smoothing in LeakyLimiter
	// func option
	atomicFunc      AtomicFunc      `json:"-"`
	deserializeFunc DeserializeFunc `json:"-"`
	serializeFunc   SerializeFunc   `json:"-"`
}

func New(name string) *baseLimiter {
	if name == "" {
		panic(LimiterError{"error 'name'"})
	}
	return &baseLimiter{
		clock: realClock{},
		name:  name,
	}
}

func (bl *baseLimiter) Name(name string) *baseLimiter {
	bl.name = name
	return bl
}

func (bl *baseLimiter) Type(tp LimiterType) *baseLimiter {
	bl.limiterType = tp
	return bl
}

func (bl *baseLimiter) CounterType() *baseLimiter {
	bl.limiterType = Counter
	return bl
}

func (bl *baseLimiter) LeakyType() *baseLimiter {
	bl.limiterType = Leaky
	return bl
}

func (bl *baseLimiter) TokenType() *baseLimiter {
	bl.limiterType = Token
	return bl
}

func (bl *baseLimiter) Storage(tp StorageType, addr string) *baseLimiter {
	bl.storageType = tp
	bl.storageAddr = addr
	return bl
}

func (bl *baseLimiter) ProcessStorage() *baseLimiter {
	bl.storageType = PROCESS_TYPE
	return bl
}

func (bl *baseLimiter) RedisStorage(addr string) *baseLimiter {
	bl.storageType = REDIS_TYPE
	bl.storageAddr = addr
	return bl
}

func (bl *baseLimiter) Unit(unit time.Duration) *baseLimiter {
	bl.unit = unit
	return bl
}

func (bl *baseLimiter) MaxReq(m int64) *baseLimiter {
	bl.maxReq = m
	return bl
}

func (bl *baseLimiter) Wait(d time.Duration) {
	bl.clock.Wait(d)
}

func (bl *baseLimiter) WaitMax(d time.Duration, md time.Duration) {
	bl.clock.MaxWait(d, md)
}

func (bl *baseLimiter) Build(ops ...Option) Limiter {
	if bl.unit <= 0 || bl.maxReq <= 0 {
		panic(LimiterError{"error 'unit' or 'maxReq' "})
	}
	ops = append(ops, WithAtomicFunc(FetchStorageAtomic), WithDeserializeFunc(JsonDeserialize), WithSerializeFunc(JsonSerialize))
	switch bl.limiterType {
	case Counter:
		ops = append(ops, WithDeserializeFunc(CounterUnmarshal), WithSerializeFunc(CounterMarshal))
		return newCounter(bl, ops...)
	case Leaky:
		return newLeaky(bl, ops...)
	case Token:
		return newToken(bl, ops...)
	default:
		panic(LimiterError{"Unknown LimiterType"})
	}
}

type Option func(*baseLimiter)

// 用于计数限流器, 为了平滑, 把限流器的单位时间分成ins间隔
func WithIns(ins int) Option {
	return func(bl *baseLimiter) {
		bl.ins = ins
	}
}

// 用于漏桶限流器, 为了平滑, 增加松弛因子
func WithSlack(slack int) Option {
	return func(bl *baseLimiter) {
		bl.slack = slack
	}
}

func WithAtomicFunc(fun AtomicFunc) Option {
	return func(bl *baseLimiter) {
		bl.atomicFunc = fun
	}
}

// 设置限流器的反序列化方法
func WithDeserializeFunc(fun DeserializeFunc) Option {
	return func(bl *baseLimiter) {
		bl.deserializeFunc = fun
	}
}

// 设置限流器的序列化方法
func WithSerializeFunc(fun SerializeFunc) Option {
	return func(bl *baseLimiter) {
		bl.serializeFunc = fun
	}
}

func JsonSerialize(l Limiter) ([]byte, error) {
	return json.Marshal(l)
}

func JsonDeserialize(data []byte, l Limiter) error {
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
