package input

import (
	"bytes"
	"testing"
)

func TestParserShiftTab(t *testing.T) {
	data := []byte("\x1b[Z")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyShiftTab {
		t.Errorf("expected KeyShiftTab, got %v", ev.Key)
	}
}

func TestParserCtrlT(t *testing.T) {
	data := []byte("\x14")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlT {
		t.Errorf("expected KeyCtrlT, got %v", ev.Key)
	}
}

func TestParserCtrlY(t *testing.T) {
	data := []byte("\x19")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlY {
		t.Errorf("expected KeyCtrlY, got %v", ev.Key)
	}
}

func TestParserCtrlG(t *testing.T) {
	data := []byte("\x07")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlG {
		t.Errorf("expected KeyCtrlG, got %v", ev.Key)
	}
}

func TestParserCtrlS(t *testing.T) {
	data := []byte("\x13")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlS {
		t.Errorf("expected KeyCtrlS, got %v", ev.Key)
	}
}

func TestParserCtrlLeft(t *testing.T) {
	data := []byte("\x1b[1;5D")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlLeft {
		t.Errorf("expected KeyCtrlLeft, got %v", ev.Key)
	}
}

func TestParserCtrlRight(t *testing.T) {
	data := []byte("\x1b[1;5C")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlRight {
		t.Errorf("expected KeyCtrlRight, got %v", ev.Key)
	}
}

func TestParserCtrlUp(t *testing.T) {
	data := []byte("\x1b[1;5A")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
if ev.Key != KeyCtrlUp {
	t.Errorf("expected KeyCtrlUp, got %v", ev.Key)
}
}

func TestParserCtrlDown(t *testing.T) {
	data := []byte("\x1b[1;5B")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
if ev.Key != KeyCtrlDown {
	t.Errorf("expected KeyCtrlDown, got %v", ev.Key)
}
}

func TestParserHome(t *testing.T) {
	data := []byte("\x1b[1~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyHome {
		t.Errorf("expected KeyHome, got %v", ev.Key)
	}
}

func TestParserEnd(t *testing.T) {
	data := []byte("\x1b[4~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyEnd {
		t.Errorf("expected KeyEnd, got %v", ev.Key)
	}
}

func TestParserHomeAlternative(t *testing.T) {
	data := []byte("\x1b[7~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyHome {
		t.Errorf("expected KeyHome for [7~, got %v", ev.Key)
	}
}

func TestParserEndAlternative(t *testing.T) {
	data := []byte("\x1b[8~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyEnd {
		t.Errorf("expected KeyEnd for [8~, got %v", ev.Key)
	}
}

func TestParserAltB(t *testing.T) {
	data := []byte("\x1bb")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlLeft {
		t.Errorf("expected KeyCtrlLeft for Alt+b, got %v", ev.Key)
	}
}

func TestParserAltF(t *testing.T) {
	data := []byte("\x1bf")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlRight {
		t.Errorf("expected KeyCtrlRight for Alt+f, got %v", ev.Key)
	}
}

func TestParserAltD(t *testing.T) {
	data := []byte("\x1bd")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlDelete {
		t.Errorf("expected KeyCtrlDelete for Alt+d, got %v", ev.Key)
	}
}

func TestParserAltBackspace(t *testing.T) {
	data := []byte("\x1b\x7f")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlBackspace {
		t.Errorf("expected KeyCtrlBackspace for Alt+Backspace, got %v", ev.Key)
	}
}

func TestParserCtrlK(t *testing.T) {
	data := []byte("\x0b")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlK {
		t.Errorf("expected KeyCtrlK, got %v", ev.Key)
	}
}

func TestParserCtrlL(t *testing.T) {
	data := []byte("\x0c")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlL {
		t.Errorf("expected KeyCtrlL, got %v", ev.Key)
	}
}

func TestParserMixedInput(t *testing.T) {
	data := []byte("hello\x1b[A\x1b[Bworld")
	p := NewParser(bytes.NewReader(data))

	events := []struct {
		key  Key
		rune rune
	}{
		{KeyRune, 'h'},
		{KeyRune, 'e'},
		{KeyRune, 'l'},
		{KeyRune, 'l'},
		{KeyRune, 'o'},
		{KeyUp, 0},
		{KeyDown, 0},
		{KeyRune, 'w'},
		{KeyRune, 'o'},
		{KeyRune, 'r'},
		{KeyRune, 'l'},
		{KeyRune, 'd'},
	}

	for i, expected := range events {
		ev, err := p.NextEvent()
		if err != nil {
			t.Fatalf("event %d: unexpected error: %v", i, err)
		}
		if ev.Key != expected.key {
			t.Errorf("event %d: expected key %v, got %v", i, expected.key, ev.Key)
		}
		if expected.rune != 0 && ev.Rune != expected.rune {
			t.Errorf("event %d: expected rune %v, got %v", i, expected.rune, ev.Rune)
		}
	}
}

func TestParserCtrlW(t *testing.T) {
	data := []byte("\x17")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlW {
		t.Errorf("expected KeyCtrlW, got %v", ev.Key)
	}
}

func TestParserCtrlU(t *testing.T) {
	data := []byte("\x15")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlU {
		t.Errorf("expected KeyCtrlU, got %v", ev.Key)
	}
}

func TestParserCtrlA(t *testing.T) {
	data := []byte("\x01")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlA {
		t.Errorf("expected KeyCtrlA, got %v", ev.Key)
	}
}

func TestParserCtrlB(t *testing.T) {
	data := []byte("\x02")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlB {
		t.Errorf("expected KeyCtrlB, got %v", ev.Key)
	}
}

func TestParserCtrlE(t *testing.T) {
	data := []byte("\x05")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlE {
		t.Errorf("expected KeyCtrlE, got %v", ev.Key)
	}
}

func TestParserCtrlF(t *testing.T) {
	data := []byte("\x06")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlF {
		t.Errorf("expected KeyCtrlF, got %v", ev.Key)
	}
}

func TestParserCtrlN(t *testing.T) {
	data := []byte("\x0e")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlN {
		t.Errorf("expected KeyCtrlN, got %v", ev.Key)
	}
}

func TestParserCtrlP(t *testing.T) {
	data := []byte("\x10")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlP {
		t.Errorf("expected KeyCtrlP, got %v", ev.Key)
	}
}

func TestParserCtrlR(t *testing.T) {
	data := []byte("\x12")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlR {
		t.Errorf("expected KeyCtrlR, got %v", ev.Key)
	}
}

func TestParserCtrlD(t *testing.T) {
	data := []byte("\x04")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyCtrlD {
		t.Errorf("expected KeyCtrlD, got %v", ev.Key)
	}
}

func TestParserDelete(t *testing.T) {
	data := []byte("\x1b[3~")
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyDelete {
		t.Errorf("expected KeyDelete, got %v", ev.Key)
	}
}

func TestParserBackspace(t *testing.T) {
	data := []byte{127}
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyBackspace {
		t.Errorf("expected KeyBackspace, got %v", ev.Key)
	}
}

func TestParserBackspaceAlternative(t *testing.T) {
	data := []byte{8}
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyBackspace {
		t.Errorf("expected KeyBackspace, got %v", ev.Key)
	}
}

func TestParserTab(t *testing.T) {
	data := []byte{'\t'}
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyTab {
		t.Errorf("expected KeyTab, got %v", ev.Key)
	}
}

func TestParserEnter(t *testing.T) {
	data := []byte{'\r'}
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyEnter {
		t.Errorf("expected KeyEnter, got %v", ev.Key)
	}
}

func TestParserEnterLF(t *testing.T) {
	data := []byte{'\n'}
	p := NewParser(bytes.NewReader(data))

	ev, err := p.NextEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Key != KeyEnter {
		t.Errorf("expected KeyEnter for LF, got %v", ev.Key)
	}
}

func TestParserUnicode(t *testing.T) {
	data := []byte("你好")
	p := NewParser(bytes.NewReader(data))

	for i, expected := range []rune{'你', '好'} {
		ev, err := p.NextEvent()
		if err != nil {
			t.Fatalf("event %d: unexpected error: %v", i, err)
		}
		if ev.Key != KeyRune {
			t.Errorf("event %d: expected KeyRune, got %v", i, ev.Key)
		}
		if ev.Rune != expected {
			t.Errorf("event %d: expected rune %q, got %q", i, expected, ev.Rune)
		}
	}
}
