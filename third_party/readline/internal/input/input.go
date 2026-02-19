package input

import (
	"bufio"
	"io"
	"time"
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
	runes  chan rune
	err    error
}

func NewParser(r io.Reader) *Parser {
	p := &Parser{
		reader: bufio.NewReader(r),
		runes:  make(chan rune, 100),
	}
	go p.fill()
	return p
}

func (p *Parser) fill() {
	defer close(p.runes)
	for {
		r, _, err := p.reader.ReadRune()
		if err != nil {
			p.err = err
			return
		}
		p.runes <- r
	}
}

func (p *Parser) NextEvent() (InputEvent, error) {
	r, ok := <-p.runes
	if !ok {
		if p.err != nil && p.err != io.EOF {
			return InputEvent{}, p.err
		}
		return InputEvent{}, io.EOF
	}
	return p.parseRune(r)
}

func (p *Parser) parseRune(r rune) (InputEvent, error) {
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
		return p.parseEscape()
	default:
		return InputEvent{Key: KeyRune, Rune: r}, nil
	}
}

func (p *Parser) readNext(timeout time.Duration) (rune, bool) {
	select {
	case r, ok := <-p.runes:
		return r, ok
	case <-time.After(timeout):
		return 0, false
	}
}

func (p *Parser) parseEscape() (InputEvent, error) {
	r, ok := p.readNext(50 * time.Millisecond)
	if !ok {
		return InputEvent{Key: KeyEsc}, nil
	}

	if r == '[' {
		r, ok = p.readNext(50 * time.Millisecond)
		if !ok {
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
			r, ok = p.readNext(50 * time.Millisecond)
			if ok && r == '~' {
				return InputEvent{Key: KeyDelete}, nil
			}
		case '1': // [1;5C (Ctrl+Right) or [1;5D (Ctrl+Left)
			r, ok = p.readNext(50 * time.Millisecond)
			if ok && r == ';' {
				r, ok = p.readNext(50 * time.Millisecond) // '5'
				r, ok = p.readNext(50 * time.Millisecond) // 'C' or 'D'
				if r == 'C' {
					return InputEvent{Key: KeyCtrlRight}, nil
				} else if r == 'D' {
					return InputEvent{Key: KeyCtrlLeft}, nil
				}
			} else if ok && r == '~' {
				return InputEvent{Key: KeyHome}, nil
			}
		case '7': // Home [7~
			r, ok = p.readNext(50 * time.Millisecond)
			if ok && r == '~' {
				return InputEvent{Key: KeyHome}, nil
			}
		case '4', '8': // End [4~ or [8~
			r, ok = p.readNext(50 * time.Millisecond)
			if ok && r == '~' {
				return InputEvent{Key: KeyEnd}, nil
			}
		}
	} else if r == 'O' {
		r, ok = p.readNext(50 * time.Millisecond)
		if !ok {
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
		}
	} else if r == 'b' {
		return InputEvent{Key: KeyCtrlLeft}, nil
	} else if r == 'f' {
		return InputEvent{Key: KeyCtrlRight}, nil
	}

	return InputEvent{Key: KeyUnknown}, nil
}
