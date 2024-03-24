package ratelimite

import (
	"testing"
)

func TestLimiter(t *testing.T) {
	lt := NewLimiter("MyLimiter", Token, 100)
	err := lt.SetRedis("127.0.0.1:6379")
	if err != nil {
		t.Error(err)
	}

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
