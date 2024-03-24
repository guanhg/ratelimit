package ratelimite

import (
	"testing"
)

func TestLeakyLimit(t *testing.T) {
	b := &base{
		name:         "testLimit",
		clock:        realClock{},
		_type:        Leaky,
		maxPerMinute: 100,
		mux:          NewAtomic(PROCESS),
	}

	// leaky限流无法应对突发流量，
	// 请求的时间间隔要大于PerRequest (maxPerMinute/time.Minute)
	// wait等待时间最大为PerRequest
	lt := newLeaky(b)
	for i := 0; i < 60; i++ {
		wait, b, err := lt.Acquired()
		if err != nil {
			t.Error(err.Error())
		}
		if !b {
			t.Errorf("Wait: %2f - [%d]", wait.Seconds(), i)
		}
		// time.Sleep(lt.PerRequest)
	}
}
