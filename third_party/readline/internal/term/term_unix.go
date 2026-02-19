//go:build !windows

package term

import (
	"io"
	"os"
	"golang.org/x/sys/unix"
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
	ws, err := unix.IoctlGetWinsize(t.fd, unix.TIOCGWINSZ)
	if err != nil {
		return 80, 24, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func (t *unixTerminal) SetRaw() (func(), error) {
	termios, err := unix.IoctlGetTermios(t.fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	oldState := *termios

	// ICRNL: Fix Ctrl-M being read as Ctrl-J
	// INLCR: Fix Ctrl-J being read as Ctrl-M
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8

	if err := unix.IoctlSetTermios(t.fd, unix.TCSETS, termios); err != nil {
		return nil, err
	}

	return func() {
		unix.IoctlSetTermios(t.fd, unix.TCSETS, &oldState)
	}, nil
}
