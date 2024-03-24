package ratelimite

import (
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	rg "github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/gomodule/redigo/redis"
)

const (
	_dataKey   = "_LimiterData"
	_lockKey   = "_LimiterLock"
	expireTime = 60 * 3 // _dataKey 3分钟过期
)

var GetEX = redis.NewScript(1, "if redis.call('EXISTS', KEYS[1]) == 1 then return redis.call('GET', KEYS[1]) else return ARGV[1] end")

type RedisAtomic struct {
	key  string
	mux  *redsync.Mutex
	conn redis.Conn
}

func (ra *RedisAtomic) SetAddr(addr string) error {
	var (
		err      error
		dialFunc = func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialConnectTimeout(10*time.Second))
		}
	)

	ra.conn, err = dialFunc()
	if err != nil {
		return err
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

	ra.mux = redsync.New(pool).NewMutex(_lockKey)
	return nil
}

func (r *RedisAtomic) SetKey(key string) {
	r.key = key
}

func (r *RedisAtomic) getType() AtomicType {
	return REDIS
}

func (r *RedisAtomic) Lock() error {
	if r.mux == nil {
		return fmt.Errorf("redis missing, check SetAddr")
	}
	return r.mux.Lock()
}
func (r *RedisAtomic) Unlock() (bool, error) {
	if r.mux == nil {
		return false, fmt.Errorf("redis missing, check SetAddr")
	}
	return r.mux.Unlock()
}

func (r *RedisAtomic) Store(data []byte) error {
	if r.conn == nil {
		return fmt.Errorf("redis missing, check SetAddr")
	}
	_, err := r.conn.Do("SET", fmt.Sprintf("%s%s", r.key, _dataKey), data, "EX", expireTime)
	return err
}

func (r *RedisAtomic) Restore() ([]byte, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("redis missing, check SetAddr")
	}
	notFound := "NotFound"
	bytes, err := redis.Bytes(GetEX.Do(r.conn, fmt.Sprintf("%s%s", r.key, _dataKey), notFound))
	if err != nil {
		return nil, err
	}
	if string(bytes) == notFound {
		return nil, nil
	}

	return bytes, nil
}
