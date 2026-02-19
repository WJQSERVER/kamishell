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
