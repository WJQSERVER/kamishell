//go:build windows

package term

import (
	"io"
	"os"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

type windowsTerminal struct {
	in  io.Reader
	out io.Writer
	hIn windows.Handle
	hOut windows.Handle
}

func newTerminal(in io.Reader, out io.Writer) (Terminal, error) {
	hIn := windows.Handle(os.Stdin.Fd())
	if f, ok := in.(*os.File); ok {
		hIn = windows.Handle(f.Fd())
	}
	hOut := windows.Handle(os.Stdout.Fd())
	if f, ok := out.(*os.File); ok {
		hOut = windows.Handle(f.Fd())
	}
	return &windowsTerminal{
		in:   in,
		out:  out,
		hIn:  hIn,
		hOut: hOut,
	}, nil
}

func (t *windowsTerminal) Read(p []byte) (n int, err error) {
	return t.in.Read(p)
}

func (t *windowsTerminal) Write(p []byte) (n int, err error) {
	return t.out.Write(p)
}

func (t *windowsTerminal) GetSize() (width, height int, err error) {
	return term.GetSize(int(t.hOut))
}

func (t *windowsTerminal) SetRaw() (func(), error) {
	oldState, err := term.MakeRaw(int(t.hIn))
	if err != nil {
		return nil, err
	}

	// MakeRaw handles input, but we still need to ensure VT processing on output
	var oldOutMode uint32
	windows.GetConsoleMode(t.hOut, &oldOutMode)
	newOutMode := oldOutMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.ENABLE_PROCESSED_OUTPUT | windows.DISABLE_NEWLINE_AUTO_RETURN
	windows.SetConsoleMode(t.hOut, newOutMode)

	// Also ensure ENABLE_VIRTUAL_TERMINAL_INPUT is set if not already by MakeRaw
	var currentInMode uint32
	windows.GetConsoleMode(t.hIn, &currentInMode)
	windows.SetConsoleMode(t.hIn, currentInMode | 0x0200)

	return func() {
		term.Restore(int(t.hIn), oldState)
		windows.SetConsoleMode(t.hOut, oldOutMode)
	}, nil
}
