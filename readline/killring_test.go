package readline

import (
	"os"
	"testing"

	"github.com/WJQSERVER/readline/internal/input"
)

func TestKillRingKillToEnd(t *testing.T) {
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
	rl.buffer.SetContent("hello world")
	rl.buffer.MoveHome()
	for j := 0; j < 5; j++ {
		rl.buffer.MoveRight()
	}
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.mu.Unlock()

	if rl.buffer.String() != "hello" {
		t.Fatalf("expected buffer 'hello', got %q", rl.buffer.String())
	}
	if len(rl.killRing) != 1 || rl.killRing[0] != " world" {
		t.Fatalf("expected killRing [' world'], got %v", rl.killRing)
	}
}

func TestKillRingKillToStart(t *testing.T) {
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
	rl.buffer.SetContent("hello world")
	rl.buffer.MoveLeft()
	rl.buffer.MoveLeft()
	rl.buffer.MoveLeft()
	rl.buffer.MoveLeft()
	rl.buffer.MoveLeft()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlU})
	rl.mu.Unlock()

	if rl.buffer.String() != "world" {
		t.Fatalf("expected buffer 'world', got %q", rl.buffer.String())
	}
	if len(rl.killRing) != 1 || rl.killRing[0] != "hello " {
		t.Fatalf("expected killRing ['hello '], got %v", rl.killRing)
	}
}

func TestKillRingBackspaceWord(t *testing.T) {
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
	rl.buffer.SetContent("hello world test")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlW})
	rl.mu.Unlock()

	if rl.buffer.String() != "hello world " {
		t.Fatalf("expected buffer 'hello world ', got %q", rl.buffer.String())
	}
	if len(rl.killRing) != 1 || rl.killRing[0] != "test" {
		t.Fatalf("expected killRing ['test'], got %v", rl.killRing)
	}
}

func TestKillRingDeleteWord(t *testing.T) {
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
	rl.buffer.SetContent("hello world test")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlDelete})
	rl.mu.Unlock()

	if rl.buffer.String() != " world test" {
		t.Fatalf("expected buffer ' world test', got %q", rl.buffer.String())
	}
	if len(rl.killRing) != 1 || rl.killRing[0] != "hello" {
		t.Fatalf("expected killRing ['hello'], got %v", rl.killRing)
	}
}

func TestKillRingYank(t *testing.T) {
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
	rl.buffer.SetContent("hello world")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("new: ")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlY})
	rl.mu.Unlock()

	if rl.buffer.String() != "new: hello world" {
		t.Fatalf("expected buffer 'new: hello world', got %q", rl.buffer.String())
	}
}

func TestKillRingYankEmptyRing(t *testing.T) {
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
	rl.buffer.SetContent("hello")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlY})
	rl.mu.Unlock()

	if rl.buffer.String() != "hello" {
		t.Fatalf("expected buffer unchanged 'hello', got %q", rl.buffer.String())
	}
}

func TestKillRingMultipleKills(t *testing.T) {
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
	rl.buffer.SetContent("one")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("two")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("three")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.mu.Unlock()

	if len(rl.killRing) != 3 {
		t.Fatalf("expected 3 items in killRing, got %d", len(rl.killRing))
	}
	if rl.killRing[0] != "one" || rl.killRing[1] != "two" || rl.killRing[2] != "three" {
		t.Fatalf("expected killRing ['one', 'two', 'three'], got %v", rl.killRing)
	}
}

func TestKillRingMaxSize(t *testing.T) {
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
	for i := 0; i < 20; i++ {
		rl.buffer.SetContent("test")
		rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	}
	rl.mu.Unlock()

	if len(rl.killRing) > killRingMaxSize {
		t.Fatalf("expected killRing size <= %d, got %d", killRingMaxSize, len(rl.killRing))
	}
}

func TestKillRingYankLatest(t *testing.T) {
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
	rl.buffer.SetContent("first")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("second")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlY})
	rl.mu.Unlock()

	if rl.buffer.String() != "second" {
		t.Fatalf("expected buffer 'second' (latest yank), got %q", rl.buffer.String())
	}
}

func TestKillRingEmptyKill(t *testing.T) {
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
	rl.buffer.SetContent("")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.mu.Unlock()

	if len(rl.killRing) != 0 {
		t.Fatalf("expected empty killRing for empty kill, got %v", rl.killRing)
	}
}

func TestKillRingUnicode(t *testing.T) {
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
	rl.buffer.SetContent("你好世界")
	rl.buffer.MoveHome()
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlK})
	rl.buffer.SetContent("新: ")
	rl.handleNormalMode(input.InputEvent{Key: input.KeyCtrlY})
	rl.mu.Unlock()

	if rl.buffer.String() != "新: 你好世界" {
		t.Fatalf("expected buffer '新: 你好世界', got %q", rl.buffer.String())
	}
}
