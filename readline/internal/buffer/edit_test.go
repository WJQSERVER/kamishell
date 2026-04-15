package buffer

import "testing"

func TestTransposeChars(t *testing.T) {
	b := NewBuffer()
	b.SetContent("abc")
	b.MoveLeft()
	b.TransposeChars()

	if b.String() != "bac" {
		t.Fatalf("expected 'bac', got %q", b.String())
	}
}

func TestTransposeCharsAtEnd(t *testing.T) {
	b := NewBuffer()
	b.SetContent("ab")
	b.TransposeChars()

	if b.String() != "ba" {
		t.Fatalf("expected 'ba', got %q", b.String())
	}
}

func TestTransposeCharsSingleChar(t *testing.T) {
	b := NewBuffer()
	b.SetContent("a")
	b.TransposeChars()

	if b.String() != "a" {
		t.Fatalf("expected 'a' (unchanged), got %q", b.String())
	}
}

func TestTransposeCharsEmpty(t *testing.T) {
	b := NewBuffer()
	b.TransposeChars()

	if b.String() != "" {
		t.Fatalf("expected '' (unchanged), got %q", b.String())
	}
}

func TestTransposeCharsUnicode(t *testing.T) {
	b := NewBuffer()
	b.SetContent("你好")
	b.TransposeChars()

	if b.String() != "好你" {
		t.Fatalf("expected '好你', got %q", b.String())
	}
}

func TestTransposeWords(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello world")
	b.MoveEnd()
	b.TransposeWords()

	if b.String() != "world hello" {
		t.Fatalf("expected 'world hello', got %q", b.String())
	}
}

func TestTransposeWordsMiddle(t *testing.T) {
	b := NewBuffer()
	b.SetContent("one two three")
	b.MoveEnd()
	b.MoveWordLeft()
	b.TransposeWords()

	if b.String() != "two one three" {
		t.Fatalf("expected 'two one three', got %q", b.String())
	}
}

func TestTransposeWordsSingleWord(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello")
	b.MoveEnd()
	b.TransposeWords()

	if b.String() != "hello" {
		t.Fatalf("expected 'hello' (unchanged), got %q", b.String())
	}
}

func TestCapitalizeWord(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello world")
	b.MoveHome()
	b.CapitalizeWord()

	if b.String() != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", b.String())
	}
	if b.Cursor() != 5 {
		t.Fatalf("expected cursor at 5, got %d", b.Cursor())
	}
}

func TestCapitalizeWordMixedCase(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hELLO world")
	b.MoveHome()
	b.CapitalizeWord()

	if b.String() != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", b.String())
	}
}

func TestCapitalizeWordAtEnd(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello")
	b.MoveEnd()
	b.CapitalizeWord()

	if b.String() != "hello" {
		t.Fatalf("expected 'hello' (unchanged), got %q", b.String())
	}
}

func TestCapitalizeWordSkipSpaces(t *testing.T) {
	b := NewBuffer()
	b.SetContent("  hello")
	b.MoveHome()
	b.CapitalizeWord()

	if b.String() != "  Hello" {
		t.Fatalf("expected '  Hello', got %q", b.String())
	}
}

func TestUppercaseWord(t *testing.T) {
	b := NewBuffer()
	b.SetContent("hello world")
	b.MoveHome()
	b.UppercaseWord()

	if b.String() != "HELLO world" {
		t.Fatalf("expected 'HELLO world', got %q", b.String())
	}
	if b.Cursor() != 5 {
		t.Fatalf("expected cursor at 5, got %d", b.Cursor())
	}
}

func TestUppercaseWordMixedCase(t *testing.T) {
	b := NewBuffer()
	b.SetContent("HeLLo WoRLd")
	b.MoveHome()
	b.UppercaseWord()

	if b.String() != "HELLO WoRLd" {
		t.Fatalf("expected 'HELLO WoRLd', got %q", b.String())
	}
}

func TestLowercaseWord(t *testing.T) {
	b := NewBuffer()
	b.SetContent("HELLO WORLD")
	b.MoveHome()
	b.LowercaseWord()

	if b.String() != "hello WORLD" {
		t.Fatalf("expected 'hello WORLD', got %q", b.String())
	}
	if b.Cursor() != 5 {
		t.Fatalf("expected cursor at 5, got %d", b.Cursor())
	}
}

func TestLowercaseWordMixedCase(t *testing.T) {
	b := NewBuffer()
	b.SetContent("HeLLo WoRLd")
	b.MoveHome()
	b.LowercaseWord()

	if b.String() != "hello WoRLd" {
		t.Fatalf("expected 'hello WoRLd', got %q", b.String())
	}
}

func TestWordOperationsUnicode(t *testing.T) {
	b := NewBuffer()
	b.SetContent("你好 世界")
	b.MoveHome()
	b.CapitalizeWord()

	if b.String() != "你好 世界" {
		t.Fatalf("expected '你好 世界' (unchanged for non-Latin), got %q", b.String())
	}
}
