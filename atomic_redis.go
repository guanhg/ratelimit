package ratelimite

import (
	"time"

	"github.com/go-redsync/redsync/v4"
	rg "github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/gomodule/redigo/redis"
)

const (
	redisLock  = "redisLimiterAtomicLock"
	expireTime = 60
)

var GetEX = redis.NewScript(1, "if redis.call('EXISTS', KEYS[1]) == 1 then return redis.call('GET', KEYS[1]) else return ARGV[1] end")

type RedisAtomic struct {
	key  string
	mutx *redsync.Mutex
	conn redis.Conn
}

func NewRedisAtomic(addr string) (*RedisAtomic, error) {
	if addr == "" {
		return nil, &LimiterError{"Redis address missing"}
	}

	dialFunc := func() (redis.Conn, error) {
		return redis.Dial("tcp", addr, redis.DialConnectTimeout(10*time.Second))
	}

	conn, err := dialFunc()
	if err != nil {
		return nil, err
	}

	pool := rg.NewPool(&redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        dialFunc,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	})

	rs := redsync.New(pool)
	mutx := rs.NewMutex(redisLock)

	return &RedisAtomic{
		conn: conn,
		mutx: mutx,
	}, nil
}

func (r *RedisAtomic) Lock() {
	if err := r.mutx.Lock(); err != nil {
		panic(err)
	}
}
func (r *RedisAtomic) Unlock() {
	if _, err := r.mutx.Unlock(); err != nil {
		panic(err)
	}
}

func (r *RedisAtomic) SetKey(key string) {
	r.key = key
}

func (r *RedisAtomic) Store(data []byte) error {
	_, err := r.conn.Do("SET", r.key, data)
	return err
}

func (r *RedisAtomic) Restore() ([]byte, error) {
	notFound := "NotFound"
	bytes, err := redis.Bytes(GetEX.Do(r.conn, r.key, notFound))
	if err != nil {
		return nil, err
	}
	if string(bytes) == notFound {
		return nil, nil
	}

	return bytes, nil
}
