package input

import (
	"bufio"
	"io"
)

type Key int

const (
	KeyUnknown Key = iota
	KeyRune
	KeyEnter
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyTab
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlK
	KeyCtrlL
	KeyCtrlN
	KeyCtrlP
	KeyCtrlR
	KeyCtrlU
	KeyCtrlW
	KeyEsc
	KeyCtrlLeft
	KeyCtrlRight
)

type InputEvent struct {
	Key  Key
	Rune rune
}

type Parser struct {
	reader *bufio.Reader
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
	}
}

func (p *Parser) NextEvent() (InputEvent, error) {
	r, _, err := p.reader.ReadRune()
	if err != nil {
		return InputEvent{}, err
	}

	switch r {
	case '\r', '\n':
		return InputEvent{Key: KeyEnter}, nil
	case 127, '\b':
		return InputEvent{Key: KeyBackspace}, nil
	case '\t':
		return InputEvent{Key: KeyTab}, nil
	case 1:
		return InputEvent{Key: KeyCtrlA}, nil
	case 2:
		return InputEvent{Key: KeyCtrlB}, nil
	case 3:
		return InputEvent{Key: KeyCtrlC}, nil
	case 4:
		return InputEvent{Key: KeyCtrlD}, nil
	case 5:
		return InputEvent{Key: KeyCtrlE}, nil
	case 6:
		return InputEvent{Key: KeyCtrlF}, nil
	case 11:
		return InputEvent{Key: KeyCtrlK}, nil
	case 12:
		return InputEvent{Key: KeyCtrlL}, nil
	case 14:
		return InputEvent{Key: KeyCtrlN}, nil
	case 16:
		return InputEvent{Key: KeyCtrlP}, nil
	case 18:
		return InputEvent{Key: KeyCtrlR}, nil
	case 21:
		return InputEvent{Key: KeyCtrlU}, nil
	case 23:
		return InputEvent{Key: KeyCtrlW}, nil
	case 27: // Escape
		// Check for escape sequences
		if p.reader.Buffered() == 0 {
			return InputEvent{Key: KeyEsc}, nil
		}
		return p.parseEscape()
	default:
		return InputEvent{Key: KeyRune, Rune: r}, nil
	}
}

func (p *Parser) parseEscape() (InputEvent, error) {
	r, _, err := p.reader.ReadRune()
	if err != nil {
		return InputEvent{Key: KeyEsc}, nil
	}

	if r == '[' {
		r, _, err = p.reader.ReadRune()
		if err != nil {
			return InputEvent{Key: KeyEsc}, nil
		}
		switch r {
		case 'A':
			return InputEvent{Key: KeyUp}, nil
		case 'B':
			return InputEvent{Key: KeyDown}, nil
		case 'C':
			return InputEvent{Key: KeyRight}, nil
		case 'D':
			return InputEvent{Key: KeyLeft}, nil
		case 'H':
			return InputEvent{Key: KeyHome}, nil
		case 'F':
			return InputEvent{Key: KeyEnd}, nil
		case '3': // Maybe Delete [3~
			r, _, _ = p.reader.ReadRune()
			if r == '~' {
				return InputEvent{Key: KeyDelete}, nil
			}
		case '1': // [1;5C (Ctrl+Right) or [1;5D (Ctrl+Left)
			// Also [1~ is Home
			r, _, _ = p.reader.ReadRune()
			if r == ';' {
				r, _, _ = p.reader.ReadRune() // '5'
				r, _, _ = p.reader.ReadRune() // 'C' or 'D'
				if r == 'C' {
					return InputEvent{Key: KeyCtrlRight}, nil
				} else if r == 'D' {
					return InputEvent{Key: KeyCtrlLeft}, nil
				}
			} else if r == '~' {
				return InputEvent{Key: KeyHome}, nil
			}
		case '7': // Home [7~
			r, _, _ = p.reader.ReadRune()
			if r == '~' {
				return InputEvent{Key: KeyHome}, nil
			}
		case '4', '8': // End [4~ or [8~
			r, _, _ = p.reader.ReadRune()
			if r == '~' {
				return InputEvent{Key: KeyEnd}, nil
			}
		}
	} else if r == 'O' {
		r, _, err = p.reader.ReadRune()
		if err != nil {
			return InputEvent{Key: KeyEsc}, nil
		}
		switch r {
		case 'H':
			return InputEvent{Key: KeyHome}, nil
		case 'F':
			return InputEvent{Key: KeyEnd}, nil
		}
	} else if r == 'b' { // Alt+b is often used as MoveWordLeft
		return InputEvent{Key: KeyCtrlLeft}, nil
	} else if r == 'f' { // Alt+f is often used as MoveWordRight
		return InputEvent{Key: KeyCtrlRight}, nil
	}

	return InputEvent{Key: KeyUnknown}, nil
}
