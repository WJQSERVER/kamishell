package term

import "io"

type Terminal interface {
	io.ReadWriter
	GetSize() (width, height int, err error)
	SetRaw() (restore func(), err error)
}

func NewTerminal(in io.Reader, out io.Writer) (Terminal, error) {
	return newTerminal(in, out)
}
