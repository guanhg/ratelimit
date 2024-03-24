package ratelimite

import (
	"sync"
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
	wg := sync.WaitGroup{}

	lt := newToken(b)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			wait, b, err := lt.Acquired()
			if err != nil {
				t.Error(err.Error())
			}
			if !b {
				t.Errorf("Wait: %f", wait.Seconds())
			}
		}(i)
	}

	wg.Wait()
}

func TestRedisTokenLimiter(t *testing.T) {
	b := &base{
		name:         "testLimit",
		clock:        realClock{},
		_type:        Token,
		maxPerMinute: 100,
		mux:          NewAtomic(PROCESS),
	}
	wg := sync.WaitGroup{}
	lt := newToken(b)
	lt.SetRedis("127.0.0.1:6379")
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			wait, b, err := lt.Acquired()
			if err != nil {
				t.Error(err)
			}
			if !b {
				t.Errorf("Wait: %f", wait.Seconds())
			}
		}(i)
	}
	wg.Wait()
}
