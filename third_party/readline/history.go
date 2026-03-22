package readline

import "sync"

type History interface {
	Append(line string)
	Get(index int) (string, bool)
	Len() int
}

type memoryHistory struct {
	lines []string
	mu    sync.RWMutex
}

func NewHistory() History {
	return &memoryHistory{
		lines: make([]string, 0),
	}
}

func (h *memoryHistory) Append(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if line == "" {
		return
	}
	// Don't append if same as last
	if len(h.lines) > 0 && h.lines[len(h.lines)-1] == line {
		return
	}
	h.lines = append(h.lines, line)
}

func (h *memoryHistory) Get(index int) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if index < 0 || index >= len(h.lines) {
		return "", false
	}
	return h.lines[index], true
}

func (h *memoryHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.lines)
}
