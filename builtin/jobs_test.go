package builtin

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompleteJobWithResultMarksFailure(t *testing.T) {
	JobsMu.Lock()
	originalJobs := Jobs
	originalNext := NextJobID
	Jobs = make(map[int]*Job)
	NextJobID = 1
	JobsMu.Unlock()

	defer func() {
		JobsMu.Lock()
		Jobs = originalJobs
		NextJobID = originalNext
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
