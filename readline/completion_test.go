package readline

import (
	"os"
	"testing"

	"github.com/WJQSERVER/readline/internal/input"
)

type testCompleter struct {
	candidates [][]rune
	length     int
}

func (tc *testCompleter) Do(line []rune, pos int) ([][]rune, int) {
	return tc.candidates, tc.length
}

func TestCompletionModeEnterMultipleCandidates(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if !rl.completionMode {
		t.Fatal("expected completionMode to be true")
	}
	if len(rl.completionCandidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(rl.completionCandidates))
	}
	if rl.completionSelected != 0 {
		t.Fatalf("expected selected index 0, got %d", rl.completionSelected)
	}
}

func TestCompletionModeTabCycles(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected initial selection 0, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 1 {
		t.Fatalf("expected selection 1 after Tab, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 2 {
		t.Fatalf("expected selection 2 after second Tab, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected selection 0 after wrap, got %d", rl.completionSelected)
	}
}

func TestCompletionModeShiftTabReverseCycles(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected initial selection 0, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyShiftTab})
	rl.mu.Unlock()

	if rl.completionSelected != 2 {
		t.Fatalf("expected selection 2 after Shift+Tab (wrap to last), got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyShiftTab})
	rl.mu.Unlock()

	if rl.completionSelected != 1 {
		t.Fatalf("expected selection 1 after second Shift+Tab, got %d", rl.completionSelected)
	}
}

func TestCompletionModeSingleCandidateAppliesDirectly(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false for single candidate")
	}
	if rl.buffer.String() != "foobar" {
		t.Fatalf("expected buffer 'foobar', got %q", rl.buffer.String())
	}
}

func TestCompletionModeCancel(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.cancelCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after cancel")
	}
	if rl.completionCandidates != nil {
		t.Fatal("expected completionCandidates to be nil after cancel")
	}
	if rl.completionReplaceLen != 0 {
		t.Fatal("expected completionReplaceLen to be 0 after cancel")
	}
}

func TestCompletionModeApply(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar"), []rune("foobaz")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.completionSelected = 1
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.buffer.String() != "foobaz" {
		t.Fatalf("expected buffer 'foobaz', got %q", rl.buffer.String())
	}
	if rl.completionMode {
		t.Fatal("expected completionMode to be false after apply")
	}
	if rl.completionReplaceLen != 0 {
		t.Fatal("expected completionReplaceLen to be 0 after apply")
	}
}

func TestCompletionModeNoCandidates(t *testing.T) {
	tc := &testCompleter{
		candidates: nil,
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false with no candidates")
	}
}

func TestCompletionModeNoCompleter(t *testing.T) {
	cfg := &Config{
		Prompt:  "> ",
		History: NewHistory(),
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false with no completer")
	}
}

func TestCompletionModeApplyTwice(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.applyCompletion()
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.buffer.String() != "foo" {
		t.Fatalf("expected buffer 'foo', got %q", rl.buffer.String())
	}
}

func TestCompletionModeApplyWithInvalidIndex(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.completionSelected = 99
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after apply with invalid index")
	}
}

func TestCompletionModeEnterApplies(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar"), []rune("foobaz")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.completionSelected = 1
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyEnter})
	rl.mu.Unlock()

	if rl.buffer.String() != "foobaz" {
		t.Fatalf("expected buffer 'foobaz' after Enter, got %q", rl.buffer.String())
	}
}

func TestCompletionModeEscCancels(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyEsc})
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after Esc")
	}
}
