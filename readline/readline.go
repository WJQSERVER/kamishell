package readline

import (
	"errors"
	"github.com/WJQSERVER/readline/internal/buffer"
	"github.com/WJQSERVER/readline/internal/input"
	"github.com/WJQSERVER/readline/internal/render"
	"github.com/WJQSERVER/readline/internal/term"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	ErrInterrupt = errors.New("interrupt")
	ErrEOF       = io.EOF
)

type Instance struct {
	cfg      *Config
	terminal term.Terminal
	buffer   *buffer.Buffer
	renderer *render.Renderer
	parser   *input.Parser
	mu       sync.Mutex

	historyIdx int // -1 means current line
	tempBuffer string
	closeOnce  sync.Once
	closed     bool
}

func NewInstance(cfg *Config) (*Instance, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Init()

	t, err := term.NewTerminal(cfg.Stdin, cfg.Stdout)
	if err != nil {
		return nil, err
	}

	return &Instance{
		cfg:        cfg,
		terminal:   t,
		buffer:     buffer.NewBuffer(),
		renderer:   render.NewRenderer(t),
		parser:     input.NewParser(t),
		historyIdx: -1,
	}, nil
}

func (i *Instance) Readline() (string, error) {
	if i.isClosed() {
		return "", ErrEOF
	}
	restore, err := i.terminal.SetRaw()
	if err != nil {
		return "", err
	}
	defer restore()

	i.mu.Lock()
	i.buffer.Clear()
	i.historyIdx = -1
	i.renderer.SetPrompt(i.cfg.Prompt)
	if runtime.GOOS == "windows" {
		time.Sleep(50 * time.Millisecond)
	}
	i.renderer.Refresh(i.buffer)
	i.mu.Unlock()

	for {
		ev, err := i.parser.NextEvent()
		if err != nil {
			return "", err
		}

		i.mu.Lock()
		stop := false
		var result string
		var resultErr error

		switch ev.Key {
		case input.KeyEnter:
			result = i.buffer.String()
			i.renderer.NewLine()
			i.cfg.History.Append(result)
			stop = true
		case input.KeyBackspace:
			i.buffer.Backspace()
		case input.KeyDelete:
			i.buffer.Delete()
		case input.KeyLeft, input.KeyCtrlB:
			i.buffer.MoveLeft()
		case input.KeyRight, input.KeyCtrlF:
			i.buffer.MoveRight()
		case input.KeyCtrlLeft:
			i.buffer.MoveWordLeft()
		case input.KeyCtrlRight:
			i.buffer.MoveWordRight()
		case input.KeyCtrlDelete:
			i.buffer.DeleteWord()
		case input.KeyHome, input.KeyCtrlA:
			i.buffer.MoveHome()
		case input.KeyEnd, input.KeyCtrlE:
			i.buffer.MoveEnd()
		case input.KeyUp, input.KeyCtrlP:
			i.handleHistory(true)
		case input.KeyDown, input.KeyCtrlN:
			i.handleHistory(false)
		case input.KeyCtrlK:
			i.buffer.KillToEnd()
		case input.KeyCtrlU:
			i.buffer.KillToStart()
		case input.KeyCtrlW, input.KeyCtrlBackspace:
			i.buffer.BackspaceWord()
		case input.KeyCtrlC:
			i.renderer.NewLine()
			resultErr = ErrInterrupt
			stop = true
		case input.KeyCtrlD:
			if i.buffer.String() == "" {
				resultErr = ErrEOF
				stop = true
			} else {
				i.buffer.Delete()
			}
		case input.KeyCtrlL:
			i.renderer.Refresh(i.buffer)
		case input.KeyRune:
			i.buffer.Insert(ev.Rune)
		case input.KeyTab:
			i.handleCompletion()
		}

		if !stop {
			i.renderer.Refresh(i.buffer)
		}
		i.mu.Unlock()

		if stop {
			return result, resultErr
		}
	}
}

// Public methods for "Notifying" the library (External control)

func (i *Instance) MoveLeft() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.MoveLeft()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) MoveRight() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.MoveRight()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) MoveHome() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.MoveHome()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) MoveEnd() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.MoveEnd()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) InsertRune(r rune) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.Insert(r)
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) Backspace() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.Backspace()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) Delete() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.buffer.Delete()
	i.renderer.Refresh(i.buffer)
}

func (i *Instance) SetPrompt(prompt string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.cfg.Prompt = prompt
	i.renderer.SetPrompt(prompt)
}

func (i *Instance) handleHistory(up bool) {
	if up {
		if i.historyIdx == -1 {
			i.tempBuffer = i.buffer.String()
			i.historyIdx = i.cfg.History.Len() - 1
		} else if i.historyIdx > 0 {
			i.historyIdx--
		} else {
			return
		}
	} else {
		if i.historyIdx == -1 {
			return
		}
		i.historyIdx++
		if i.historyIdx >= i.cfg.History.Len() {
			i.historyIdx = -1
			i.buffer.SetContent(i.tempBuffer)
			return
		}
	}

	if line, ok := i.cfg.History.Get(i.historyIdx); ok {
		i.buffer.SetContent(line)
	}
}

func (i *Instance) handleCompletion() {
	if i.cfg.Completer == nil {
		return
	}
	candidates, length := i.cfg.Completer.Do(i.buffer.Runes(), i.buffer.Cursor())
	if len(candidates) == 1 {
		// Single match, just insert it
		suffix := candidates[0][length:]
		for _, r := range suffix {
			i.buffer.Insert(r)
		}
	}
}

func (i *Instance) Close() error {
	i.closeOnce.Do(func() {
		i.mu.Lock()
		i.closed = true
		i.mu.Unlock()
		if i.parser != nil {
			_ = i.parser.Close()
		}
		if i.cfg != nil && i.cfg.Stdin != nil && i.cfg.Stdin != os.Stdin {
			_ = i.cfg.Stdin.Close()
		}
	})
	return nil
}

func (i *Instance) NotifyKeyPress(k string) {
	if i.isClosed() {
		return
	}
	switch k {
	case "Left":
		i.MoveLeft()
	case "Right":
		i.MoveRight()
	case "Up":
		i.mu.Lock()
		i.handleHistory(true)
		i.renderer.Refresh(i.buffer)
		i.mu.Unlock()
	case "Down":
		i.mu.Lock()
		i.handleHistory(false)
		i.renderer.Refresh(i.buffer)
		i.mu.Unlock()
	case "Home":
		i.MoveHome()
	case "End":
		i.MoveEnd()
	case "Backspace":
		i.Backspace()
	case "Delete":
		i.Delete()
	}
}

func (i *Instance) isClosed() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.closed
}
