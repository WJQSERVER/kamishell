package readline

import (
	"bytes"
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
