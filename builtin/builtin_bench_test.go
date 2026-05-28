package builtin

import (
	"sync"
	"testing"
)

func BenchmarkRegisterJobSequential(b *testing.B) {
	JobsMu.Lock()
	origJobs := Jobs
	origNext := nextJobID.Load()
	JobsMu.Unlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JobsMu.Lock()
		Jobs = make(map[int]*Job)
		nextJobID.Store(0)
		JobsMu.Unlock()

		for range 64 {
			RegisterJob("bench-cmd", nil)
		}
	}

	JobsMu.Lock()
	Jobs = origJobs
	nextJobID.Store(origNext)
	JobsMu.Unlock()
}

func BenchmarkRegisterJobParallel(b *testing.B) {
	JobsMu.Lock()
	origJobs := Jobs
	origNext := nextJobID.Load()
	JobsMu.Unlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JobsMu.Lock()
		Jobs = make(map[int]*Job)
		nextJobID.Store(0)
		JobsMu.Unlock()

		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 8 {
					RegisterJob("bench-cmd", nil)
				}
			})
		}
		wg.Wait()
	}

	JobsMu.Lock()
	Jobs = origJobs
	nextJobID.Store(origNext)
	JobsMu.Unlock()
}
