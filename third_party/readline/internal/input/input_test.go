package input

import (
	"bytes"
	"testing"
)

func TestParser(t *testing.T) {
	data := []byte("a\r\x1b[A\x1b[3~")
	p := NewParser(bytes.NewReader(data))

	ev, _ := p.NextEvent()
	if ev.Key != KeyRune || ev.Rune != 'a' {
		t.Errorf("expected 'a', got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyEnter {
		t.Errorf("expected Enter, got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyUp {
		t.Errorf("expected Up, got %v", ev)
	}

	ev, _ = p.NextEvent()
	if ev.Key != KeyDelete {
		t.Errorf("expected Delete, got %v", ev)
	}
}

func TestParser_CtrlDelete(t *testing.T) {
	data := []byte("\x1b[3;5~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlDelete {
		t.Errorf("expected KeyCtrlDelete, got %v", ev.Key)
	}
}

func TestParserCloseUnblocksNextEvent(t *testing.T) {
	p := NewParser(bytes.NewReader(nil))
	if err := p.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
	_, err := p.NextEvent()
	if err == nil {
		t.Fatal("expected EOF after parser close")
	}
}
