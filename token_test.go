package ratelimite

import (
	"testing"
)

func TestProcessTokenLimit(t *testing.T) {
	lt := NewTokenLimiter(100)
	for i := 0; i < 100; i++ {
		go func(j int) {
			wait, b := lt.Take()
			if !b {
				t.Errorf("Wait: %f", wait.Seconds())
			}
		}(i)
	}

}
