//go:build !windows

package term

import (
	"io"
	"os"
	"golang.org/x/term"
)

type unixTerminal struct {
	in  io.Reader
	out io.Writer
	fd  int
}

func newTerminal(in io.Reader, out io.Writer) (Terminal, error) {
	fd := int(os.Stdin.Fd())
	if f, ok := in.(*os.File); ok {
		fd = int(f.Fd())
	}
	return &unixTerminal{
		in:  in,
		out: out,
		fd:  fd,
	}, nil
}

func (t *unixTerminal) Read(p []byte) (n int, err error) {
	return t.in.Read(p)
}

func (t *unixTerminal) Write(p []byte) (n int, err error) {
	return t.out.Write(p)
}

func (t *unixTerminal) GetSize() (width, height int, err error) {
	return term.GetSize(t.fd)
}

func (t *unixTerminal) SetRaw() (func(), error) {
	oldState, err := term.MakeRaw(t.fd)
	if err != nil {
		return nil, err
	}
	return func() {
		term.Restore(t.fd, oldState)
	}, nil
}
