package builtin

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestCompleteJobWithResultMarksFailure(t *testing.T) {
	JobsMu.Lock()
	originalJobs := Jobs
	originalNext := nextJobID.Load()
	Jobs = make(map[int]*Job)
	nextJobID.Store(0)
	JobsMu.Unlock()

	defer func() {
		JobsMu.Lock()
		Jobs = originalJobs
		nextJobID.Store(originalNext)
		JobsMu.Unlock()
	}()

	id := RegisterJob("sleep 1")
	CompleteJobWithResult(id, false, "exit status 1")

	stdout := &bytes.Buffer{}
	code := JobsCmd(nil, nil, bytes.NewReader(nil), stdout, bytes.NewBuffer(nil))
	if code != 0 {
		t.Fatalf("expected jobs command to succeed, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Failed") {
		t.Fatalf("expected failed status in jobs output, got %q", output)
	}
	if !strings.Contains(output, "exit status 1") {
		t.Fatalf("expected error detail in jobs output, got %q", output)
	}
}

func TestRegisterJobConcurrentUniqueness(t *testing.T) {
	JobsMu.Lock()
	originalJobs := Jobs
	originalNext := nextJobID.Load()
	Jobs = make(map[int]*Job)
	nextJobID.Store(0)
	JobsMu.Unlock()

	defer func() {
		JobsMu.Lock()
		Jobs = originalJobs
		nextJobID.Store(originalNext)
		JobsMu.Unlock()
	}()

	const goroutines = 16
	const perGoroutine = 64
	ids := make(chan int, goroutines*perGoroutine)

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range perGoroutine {
				ids <- RegisterJob("test-cmd")
			}
		})
	}
	wg.Wait()
	close(ids)

	seen := make(map[int]bool, goroutines*perGoroutine)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate job ID: %d", id)
		}
		seen[id] = true
	}

	expected := goroutines * perGoroutine
	if len(seen) != expected {
		t.Fatalf("expected %d unique IDs, got %d", expected, len(seen))
	}
}
