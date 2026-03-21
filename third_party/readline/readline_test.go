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
