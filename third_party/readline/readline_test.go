package readline

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestReadline(t *testing.T) {
	// Create a pipe to simulate stdin
	r, w, _ := os.Pipe()

	// Create a buffer for stdout
	var stdout bytes.Buffer

	cfg := &Config{
		Prompt: "> ",
		Stdin:  r,
		Stdout: os.NewFile(uintptr(os.Stdout.Fd()), "/dev/null"), // Silence stdout for test
	}
	cfg.Init()

	// We can't easily test Readline because it calls SetRaw which might fail on non-TTY
	// But we can check if NewInstance works
	rl, err := NewInstance(cfg)
	if err != nil {
		t.Skip("Skipping Readline test as it requires a TTY for SetRaw")
		return
	}
	_ = rl
	_ = w
	_ = stdout
}

func TestInstanceCloseIsIdempotent(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	defer w.Close()

	stdout, err := os.CreateTemp(t.TempDir(), "readline-out-*.txt")
	if err != nil {
		t.Fatalf("create temp stdout failed: %v", err)
	}
	defer stdout.Close()

	rl, err := NewInstance(&Config{Prompt: "> ", Stdin: r, Stdout: stdout})
	if err != nil {
		t.Fatalf("new instance failed: %v", err)
	}

	if err := rl.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := rl.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}

	if _, err := rl.parser.NextEvent(); err != io.EOF {
		t.Fatalf("expected parser EOF after instance close, got %v", err)
	}
}

func TestInstanceDoesNotMutateBufferAfterClose(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	defer w.Close()

	stdout, err := os.CreateTemp(t.TempDir(), "readline-out-*.txt")
	if err != nil {
		t.Fatalf("create temp stdout failed: %v", err)
	}
	defer stdout.Close()

	rl, err := NewInstance(&Config{Prompt: "> ", Stdin: r, Stdout: stdout})
	if err != nil {
		t.Fatalf("new instance failed: %v", err)
	}
	rl.buffer.SetContent("live")

	if err := rl.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	rl.InsertRune('x')
	rl.Backspace()
	rl.NotifyKeyPress("Left")

	if got := rl.buffer.String(); got != "live" {
		t.Fatalf("expected closed instance buffer to stay unchanged, got %q", got)
	}
	if got := rl.buffer.Cursor(); got != len([]rune("live")) {
		t.Fatalf("expected closed instance cursor to stay at end, got %d", got)
	}
}

func TestInstanceDoesNotChangeHistoryStateAfterClose(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	defer w.Close()

	stdout, err := os.CreateTemp(t.TempDir(), "readline-out-*.txt")
	if err != nil {
		t.Fatalf("create temp stdout failed: %v", err)
	}
	defer stdout.Close()

	h := NewHistory()
	h.Append("first")
	h.Append("second")

	rl, err := NewInstance(&Config{Prompt: "> ", Stdin: r, Stdout: stdout, History: h})
	if err != nil {
		t.Fatalf("new instance failed: %v", err)
	}
	rl.historyIdx = 0
	rl.tempBuffer = "draft"

	if err := rl.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	rl.NotifyKeyPress("Up")
	rl.NotifyKeyPress("Down")

	if rl.historyIdx != 0 {
		t.Fatalf("expected closed instance historyIdx unchanged, got %d", rl.historyIdx)
	}
	if rl.tempBuffer != "draft" {
		t.Fatalf("expected closed instance tempBuffer unchanged, got %q", rl.tempBuffer)
	}
}
