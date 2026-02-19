//go:build windows

package term

import (
	"io"
	"os"
	"golang.org/x/sys/windows"
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
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(t.hOut, &info); err != nil {
		return 80, 24, err
	}
	return int(info.Window.Right - info.Window.Left + 1), int(info.Window.Bottom - info.Window.Top + 1), nil
}

func (t *windowsTerminal) SetRaw() (func(), error) {
	var oldInMode, oldOutMode uint32
	if err := windows.GetConsoleMode(t.hIn, &oldInMode); err != nil {
		return nil, err
	}
	if err := windows.GetConsoleMode(t.hOut, &oldOutMode); err != nil {
		return nil, err
	}

	// Raw mode: disable echo, line processing, etc.
	newInMode := oldInMode &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_INPUT)
	// Enable virtual terminal processing for ANSI sequences
	newOutMode := oldOutMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.DISABLE_NEWLINE_AUTO_RETURN

	windows.SetConsoleMode(t.hIn, newInMode)
	windows.SetConsoleMode(t.hOut, newOutMode)

	return func() {
		windows.SetConsoleMode(t.hIn, oldInMode)
		windows.SetConsoleMode(t.hOut, oldOutMode)
	}, nil
}
