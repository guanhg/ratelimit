package ratelimite

import (
	"sync"
	"testing"
	"time"
)

func TestCount(t *testing.T) {
	lt := NewCountLimiter(199)

	gs := 20
	ch := make(chan struct{}, gs)
	var wg sync.WaitGroup
	wg.Add(10 * gs)
	for j := 0; j < 10; j++ {
		for i := 0; i < gs; i++ {
			ch <- struct{}{}
			go func(j int) {
				defer func() {
					wg.Done()
					<-ch
				}()
				wait, b := lt.Take()
				if !b {
					t.Errorf("Wait: %f", wait.Seconds())
				}
				time.Sleep(time.Millisecond * 100)
			}(i)
		}
	}
	wg.Wait()
}
