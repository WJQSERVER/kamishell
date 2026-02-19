package readline

type History interface {
	Append(line string)
	Get(index int) (string, bool)
	Len() int
}

type memoryHistory struct {
	lines []string
}

func NewHistory() History {
	return &memoryHistory{
		lines: make([]string, 0),
	}
}

func (h *memoryHistory) Append(line string) {
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
	if index < 0 || index >= len(h.lines) {
		return "", false
	}
	return h.lines[index], true
}

func (h *memoryHistory) Len() int {
	return len(h.lines)
}
