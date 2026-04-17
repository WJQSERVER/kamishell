package term

import (
	"bytes"
	"io"
	"os"
	"testing"

	"golang.org/x/term"
)

func TestNewTerminalCreatesUnixTerminal(t *testing.T) {
	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	if term == nil {
		t.Fatal("expected non-nil terminal")
	}

	var buf [1]byte
	n, err := term.Read(buf[:])
	if err != io.EOF {
		t.Fatalf("expected EOF on empty reader, got n=%d err=%v", n, err)
	}
}

func TestNewTerminalWithFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "term-test-*.tmp")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer f.Close()

	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(f, w)
	if err != nil {
		t.Fatalf("NewTerminal with file failed: %v", err)
	}

	if term == nil {
		t.Fatal("expected non-nil terminal with file")
	}
}

func TestTerminalWrite(t *testing.T) {
	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	data := []byte("hello world")
	n, err := term.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected n=%d, got %d", len(data), n)
	}

	if w.String() != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", w.String())
	}
}

func TestTerminalReadFromReader(t *testing.T) {
	r := bytes.NewBufferString("test data")
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	var buf [10]byte
	n, err := term.Read(buf[:])
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 9 {
		t.Fatalf("expected n=9, got %d", n)
	}
	if string(buf[:n]) != "test data" {
		t.Fatalf("expected %q, got %q", "test data", string(buf[:n]))
	}
}

func TestTerminalReadEmpty(t *testing.T) {
	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	var buf [1]byte
	_, err = term.Read(buf[:])
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func requiresTTY(t *testing.T) bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func TestTerminalGetSize(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("GetSize requires a TTY")
	}

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	width, height, err := term.GetSize()
	if err != nil {
		t.Fatalf("GetSize failed: %v", err)
	}

	if width < 0 || height < 0 {
		t.Fatalf("GetSize returned negative values: width=%d height=%d", width, height)
	}
}

func TestTerminalGetSizeWithStdin(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("GetSize requires a TTY")
	}

	term, err := NewTerminal(os.Stdin, os.Stdout)
	if err != nil {
		t.Fatalf("NewTerminal with stdin/stdout failed: %v", err)
	}

	width, height, err := term.GetSize()
	if err != nil {
		t.Fatalf("GetSize on stdin/stdout failed: %v", err)
	}

	if width < 0 || height < 0 {
		t.Fatalf("GetSize returned negative values: width=%d height=%d", width, height)
	}
}

func TestTerminalSetRawReturnsRestoreFunc(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	restore, err := term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}

	if restore == nil {
		t.Fatal("expected non-nil restore function")
	}

	restore()
}

func TestTerminalRestoreIdempotent(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	restore, err := term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}

	restore()
	restore()
	restore()
}

func TestTerminalSetRawAndWrite(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	_, err = term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}

	data := []byte("after raw mode")
	n, err := term.Write(data)
	if err != nil {
		t.Fatalf("Write after SetRaw failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("expected n=%d, got %d", len(data), n)
	}
}

func TestTerminalSetRawAndRead(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r := bytes.NewReader([]byte("test"))
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	_, err = term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}

	var buf [10]byte
	n, err := term.Read(buf[:])
	if err != nil && err != io.EOF {
		t.Fatalf("Read after SetRaw failed: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected n=4, got %d", n)
	}
	if string(buf[:n]) != "test" {
		t.Fatalf("expected %q, got %q", "test", string(buf[:n]))
	}
}

func TestTerminalInterface(t *testing.T) {
	var term Terminal

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	var err error
	term, err = NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	{
		var _ io.Reader = term
		var _ io.Writer = term
		var _ io.ReadWriter = term
	}

	if term == nil {
		t.Fatal("expected non-nil terminal implementing io.ReadWriter")
	}
}

func TestTerminalMultipleSetRaw(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r := bytes.NewReader(nil)
	w := bytes.NewBuffer(nil)

	term, err := NewTerminal(r, w)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	restore1, err := term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}

	restore2, err := term.SetRaw()
	if err != nil {
		t.Fatalf("Second SetRaw failed: %v", err)
	}

	restore1()
	restore2()
}

func TestTerminalSetRawWithPipedInput(t *testing.T) {
	if !requiresTTY(t) {
		t.Skip("SetRaw requires a TTY")
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	defer r.Close()
	defer w.Close()

	out := bytes.NewBuffer(nil)
	term, err := NewTerminal(r, out)
	if err != nil {
		t.Fatalf("NewTerminal failed: %v", err)
	}

	restore, err := term.SetRaw()
	if err != nil {
		t.Fatalf("SetRaw failed: %v", err)
	}
	defer restore()

	w.Write([]byte("data"))
	var buf [10]byte
	r.Read(buf[:])
}

func TestTerminalNewWithNilReaderDoesNotPanicOnWrite(t *testing.T) {
	w := bytes.NewBuffer(nil)
	term, err := NewTerminal(nil, w)
	if err != nil {
		t.Fatalf("NewTerminal with nil reader failed: %v", err)
	}

	n, err := term.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write should not fail with nil reader: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected n=4, got %d", n)
	}
}
