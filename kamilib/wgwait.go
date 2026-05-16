package kamilib

import (
	"fmt"
	"sync"
	"time"
)

// WaitTimeout waits for a sync.WaitGroup to finish with a timeout.
// Returns nil if the WaitGroup completes before the timeout,
// or an error if the timeout is reached.
func WaitTimeout(wg *sync.WaitGroup, secs int64) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(time.Duration(secs) * time.Second):
		return fmt.Errorf("WaitGroup timeout")
	}
}
