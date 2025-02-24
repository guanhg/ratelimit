package ratelimite

import (
	"testing"
	"time"
)

func TestLeakyLimit(t *testing.T) {
	// leaky限流无法应对突发流量，
	// 请求的时间间隔要大于PerRate (thresholdInMs/threshold)
	// wait等待时间最大为PerRate
	lt := NewLeakyLimiter(100)
	for i := 0; i < 110; i++ {
		wait, b := lt.Take()
		if !b {
			t.Errorf("Wait: %d - [%d]", wait.Milliseconds(), i)
		}
		time.Sleep(time.Duration(lt.PerRate() * float64(time.Millisecond)))
	}
}
