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
	"strings"
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

	historyIdx int
	tempBuffer string
	closeOnce  sync.Once
	closed     bool

	searchMode      bool
	searchQuery     *buffer.Buffer
	searchResult    string
	searchResultIdx int

	completionMode       bool
	completionCandidates [][]rune
	completionSelected   int
	completionReplaceLen int

	killRing []string
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
		cfg:         cfg,
		terminal:    t,
		buffer:      buffer.NewBuffer(),
		renderer:    render.NewRenderer(t),
		parser:      input.NewParser(t),
		historyIdx:  -1,
		searchQuery: buffer.NewBuffer(),
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
	i.searchMode = false
	i.searchQuery.Clear()
	i.searchResult = ""
	i.searchResultIdx = -1
	i.completionMode = false
	i.completionCandidates = nil
	i.completionSelected = 0
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

		if i.searchMode {
			stop, result, resultErr = i.handleSearchMode(ev)
		} else {
			stop, result, resultErr = i.handleNormalMode(ev)
		}

		if !stop {
			if i.searchMode {
				i.renderer.RefreshSearch(i.searchQuery, i.searchResult)
			} else if i.completionMode && len(i.completionCandidates) > 1 {
				i.renderer.RefreshWithCompletion(i.buffer, i.completionCandidates, i.completionSelected)
			} else {
				i.renderer.Refresh(i.buffer)
			}
		}
		i.mu.Unlock()

		if stop {
			return result, resultErr
		}
	}
}

func (i *Instance) handleNormalMode(ev input.InputEvent) (bool, string, error) {
	stop := false
	var result string
	var resultErr error

	if i.completionMode && len(i.completionCandidates) > 1 {
		return i.handleCompletionMode(ev)
	}

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
		killed := i.buffer.DeleteWord()
		if killed != "" {
			i.pushKillRing(killed)
		}
	case input.KeyHome, input.KeyCtrlA:
		i.buffer.MoveHome()
	case input.KeyEnd, input.KeyCtrlE:
		i.buffer.MoveEnd()
	case input.KeyUp, input.KeyCtrlP:
		i.handleHistory(true)
	case input.KeyDown, input.KeyCtrlN:
		i.handleHistory(false)
	case input.KeyCtrlK:
		killed := i.buffer.KillToEnd()
		if killed != "" {
			i.pushKillRing(killed)
		}
	case input.KeyCtrlU:
		killed := i.buffer.KillToStart()
		if killed != "" {
			i.pushKillRing(killed)
		}
	case input.KeyCtrlW, input.KeyCtrlBackspace:
		killed := i.buffer.BackspaceWord()
		if killed != "" {
			i.pushKillRing(killed)
		}
	case input.KeyCtrlY:
		i.yank()
	case input.KeyCtrlT:
		i.buffer.TransposeChars()
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
	case input.KeyCtrlR:
		i.startSearch()
	case input.KeyRune:
		i.buffer.Insert(ev.Rune)
	case input.KeyTab:
		i.handleCompletion()
	}

	return stop, result, resultErr
}

func (i *Instance) handleCompletionMode(ev input.InputEvent) (bool, string, error) {
	stop := false
	var result string
	var resultErr error

	switch ev.Key {
	case input.KeyTab:
		i.completionSelected = (i.completionSelected + 1) % len(i.completionCandidates)
	case input.KeyShiftTab:
		i.completionSelected--
		if i.completionSelected < 0 {
			i.completionSelected = len(i.completionCandidates) - 1
		}
	case input.KeyEnter:
		i.applyCompletion()
	case input.KeyEsc, input.KeyCtrlC, input.KeyCtrlG:
		i.cancelCompletion()
	case input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown:
		i.cancelCompletion()
		return i.handleNormalMode(ev)
	case input.KeyRune:
		i.applyCompletion()
		i.buffer.Insert(ev.Rune)
		i.cancelCompletion()
	default:
		i.cancelCompletion()
	}

	return stop, result, resultErr
}

func (i *Instance) startSearch() {
	i.searchMode = true
	i.searchQuery.Clear()
	i.searchResult = ""
	i.searchResultIdx = -1
}

func (i *Instance) handleSearchMode(ev input.InputEvent) (bool, string, error) {
	stop := false
	var result string
	var resultErr error

	switch ev.Key {
	case input.KeyEnter:
		if i.searchResult != "" {
			i.buffer.SetContent(i.searchResult)
		}
		i.searchMode = false
		i.renderer.NewLine()
		i.cfg.History.Append(i.buffer.String())
		stop = true
		result = i.buffer.String()
	case input.KeyCtrlC, input.KeyCtrlG:
		i.searchMode = false
		i.searchQuery.Clear()
		i.searchResult = ""
		i.renderer.Refresh(i.buffer)
	case input.KeyEsc:
		i.searchMode = false
		i.searchQuery.Clear()
		i.searchResult = ""
	case input.KeyBackspace:
		i.searchQuery.Backspace()
		i.updateSearchResult()
	case input.KeyCtrlR:
		i.searchPrevMatch()
	case input.KeyCtrlS:
		i.searchNextMatch()
	case input.KeyRune:
		i.searchQuery.Insert(ev.Rune)
		i.updateSearchResult()
	default:
		if i.searchResult != "" {
			i.buffer.SetContent(i.searchResult)
		}
		i.searchMode = false
	}

	return stop, result, resultErr
}

func (i *Instance) updateSearchResult() {
	query := i.searchQuery.String()
	if query == "" {
		i.searchResult = ""
		i.searchResultIdx = -1
		return
	}

	for idx := i.cfg.History.Len() - 1; idx >= 0; idx-- {
		line, ok := i.cfg.History.Get(idx)
		if !ok {
			continue
		}
		if strings.Contains(line, query) {
			i.searchResult = line
			i.searchResultIdx = idx
			return
		}
	}
	i.searchResult = ""
	i.searchResultIdx = -1
}

func (i *Instance) searchPrevMatch() {
	query := i.searchQuery.String()
	if query == "" {
		return
	}

	histLen := i.cfg.History.Len()
	if histLen == 0 {
		return
	}

	startIdx := i.searchResultIdx - 1
	if startIdx < 0 {
		startIdx = histLen - 1
	}

	for idx := startIdx; idx >= 0; idx-- {
		line, ok := i.cfg.History.Get(idx)
		if !ok {
			continue
		}
		if strings.Contains(line, query) {
			i.searchResult = line
			i.searchResultIdx = idx
			return
		}
	}
}

func (i *Instance) searchNextMatch() {
	query := i.searchQuery.String()
	if query == "" {
		return
	}

	histLen := i.cfg.History.Len()
	if histLen == 0 {
		return
	}

	startIdx := i.searchResultIdx + 1
	if startIdx >= histLen {
		startIdx = 0
	}

	for idx := startIdx; idx < histLen; idx++ {
		line, ok := i.cfg.History.Get(idx)
		if !ok {
			continue
		}
		if strings.Contains(line, query) {
			i.searchResult = line
			i.searchResultIdx = idx
			return
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
	if len(candidates) == 0 {
		return
	}

	if len(candidates) == 1 {
		suffix := candidates[0][length:]
		for _, r := range suffix {
			i.buffer.Insert(r)
		}
		return
	}

	i.completionMode = true
	i.completionCandidates = candidates
	i.completionSelected = 0
	i.completionReplaceLen = length
}

func (i *Instance) applyCompletion() {
	if !i.completionMode || len(i.completionCandidates) == 0 {
		return
	}

	if i.completionSelected < 0 || i.completionSelected >= len(i.completionCandidates) {
		i.cancelCompletion()
		return
	}

	candidate := i.completionCandidates[i.completionSelected]
	suffix := candidate[i.completionReplaceLen:]
	for _, r := range suffix {
		i.buffer.Insert(r)
	}

	i.completionMode = false
	i.completionCandidates = nil
	i.completionSelected = 0
	i.completionReplaceLen = 0
}

func (i *Instance) cancelCompletion() {
	i.completionMode = false
	i.completionCandidates = nil
	i.completionSelected = 0
	i.completionReplaceLen = 0
}

const killRingMaxSize = 16

func (i *Instance) pushKillRing(text string) {
	if text == "" {
		return
	}
	i.killRing = append(i.killRing, text)
	if len(i.killRing) > killRingMaxSize {
		i.killRing = i.killRing[1:]
	}
}

func (i *Instance) yank() {
	if len(i.killRing) == 0 {
		return
	}
	text := i.killRing[len(i.killRing)-1]
	for _, r := range text {
		i.buffer.Insert(r)
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
