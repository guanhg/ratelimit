package ratelimite

import (
	"testing"
)

func TestProcessTokenLimit(t *testing.T) {
	b := &base{
		name:         "testLimit",
		clock:        realClock{},
		_type:        Token,
		maxPerMinute: 100,
		mux:          NewAtomic(PROCESS),
	}

	lt := newToken(b)
	for i := 0; i < 100; i++ {
		go func(j int) {
			wait, b, err := lt.Acquired()
			if err != nil {
				t.Error(err.Error())
			}
			if !b {
				t.Errorf("Wait: %f", wait.Seconds())
			}
		}(i)
	}

}

func TestRedisTokenLimiter(t *testing.T) {
	b := &base{
		name:         "testLimit",
		clock:        realClock{},
		_type:        Token,
		maxPerMinute: 100,
		mux:          NewAtomic(PROCESS),
	}
	lt := newToken(b)
	lt.SetRedis("127.0.0.1:6379")
	for i := 0; i < 100; i++ {
		go func(j int) {
			wait, b, err := lt.Acquired()
			if err != nil {
				t.Error(err)
			}
			if !b {
				t.Errorf("Wait: %f", wait.Seconds())
			}
		}(i)
	}
}
