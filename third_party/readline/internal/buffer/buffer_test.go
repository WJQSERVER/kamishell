package buffer

import "testing"

func TestBuffer(t *testing.T) {
	b := NewBuffer()
	b.Insert('h')
	b.Insert('e')
	b.Insert('l')
	b.Insert('l')
	b.Insert('o')
	if b.String() != "hello" {
		t.Errorf("expected hello, got %s", b.String())
	}
	if b.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", b.Cursor())
	}

	b.MoveLeft()
	b.Insert(' ')
	if b.String() != "hell o" {
		t.Errorf("expected 'hell o', got '%s'", b.String())
	}

	b.Backspace()
	if b.String() != "hello" {
		t.Errorf("expected hello, got %s", b.String())
	}
}

func TestUnicode(t *testing.T) {
	b := NewBuffer()
	b.Insert('你')
	b.Insert('好')
	if b.String() != "你好" {
		t.Errorf("expected 你好, got %s", b.String())
	}
	if b.FullWidth() != 4 {
		t.Errorf("expected width 4, got %d", b.FullWidth())
	}
}

func TestWordMovement(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello world  test")
	// Cursor is at end (17)

	b.MoveWordLeft() // Should skip trailing space and move to start of "test" (13)
	if b.Cursor() != 13 {
		t.Errorf("expected cursor 13, got %d", b.Cursor())
	}

	b.MoveWordLeft() // Should move to start of "world" (6)
	if b.Cursor() != 6 {
		t.Errorf("expected cursor 6, got %d", b.Cursor())
	}

	b.MoveWordRight() // Should move to end of "world" (11)
	if b.Cursor() != 11 {
		t.Errorf("expected cursor 11, got %d", b.Cursor())
	}
}
