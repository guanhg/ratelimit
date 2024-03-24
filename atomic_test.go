package ratelimite

import (
	"sync"
	"testing"
	"time"
)

func TestProcessAtomic(t *testing.T) {
	mux := NewAtomic(PROCESS)

	var res []int
	expected := 1000
	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			mux.Lock()
			defer mux.Unlock()
			res = append(res, j)
		}(i)
	}

	wg.Wait()

	if len(res) != expected {
		t.Errorf("Expect: %d, Result: %d", expected, len(res))
	}
}

func TestRedisAtomic(t *testing.T) {
	mux, ok := NewAtomic(REDIS).(*RedisAtomic)
	if !ok {
		t.Error("NewAtopmic(REDIS) error")
	}
	mux.SetAddr("127.0.0.1:6379")

	var res []int
	expected := 1000

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			lc := 0
		trylock: // 尝试3次获取锁，每次间隔1s
			if err := mux.Lock(); err != nil {
				if lc < 3 {
					time.Sleep(time.Second)
					goto trylock
				}
				return
			}
			defer mux.Unlock()
			res = append(res, j)
		}(i)
	}

	wg.Wait()

	if len(res) != expected {
		t.Errorf("Expect: %d, Result: %d", expected, len(res))
	}
}
