package render

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/WJQSERVER/readline/internal/buffer"
	"github.com/mattn/go-runewidth"
)

type sizedWriter interface {
	GetSize() (width, height int, err error)
}

type Renderer struct {
	out       io.Writer
	prompt    string
	lastWidth int
	lastRows  int
}

func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{
		out: out,
	}
}

func (r *Renderer) SetPrompt(prompt string) {
	r.prompt = prompt
}

func (r *Renderer) Refresh(b *buffer.Buffer) error {
	currentWidth := b.FullWidth()
	cursorPos := b.DisplayWidth(b.Cursor())

	// Strip ANSI sequences to calculate true visual width of the prompt
	visualPrompt := stripANSI(r.prompt)
	promptWidth := runewidth.StringWidth(visualPrompt)
	termWidth := r.getTerminalWidth()
	currentRows := rowsForWidth(promptWidth+currentWidth, termWidth)
	cursorRow, cursorCol := cursorPosition(promptWidth+cursorPos, termWidth)

	var out bytes.Buffer

	// Hide cursor to prevent jitter
	out.WriteString("\x1b[?25l")

	if r.lastRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", r.lastRows-1)
	}
	for i := 0; i < max(1, r.lastRows); i++ {
		out.WriteString("\r\x1b[2K")
		if i < max(1, r.lastRows)-1 {
			out.WriteString("\x1b[1B")
		}
	}
	if r.lastRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", r.lastRows-1)
	}

	// Move to column 1 and redraw prompt and content.
	fmt.Fprintf(&out, "\x1b[1G%s%s", r.prompt, b.String())
	if currentRows > 1 {
		segments := wrapVisualSegments(r.prompt, promptWidth, b.String(), termWidth)
		if len(segments) > 0 {
			out.WriteString("\r")
			out.WriteString(strings.Join(segments, "\r\n"))
		}
	}
	out.WriteString("\x1b[K")

	if currentRows > 1 {
		rowsDown := currentRows - 1 - cursorRow
		if rowsDown > 0 {
			fmt.Fprintf(&out, "\x1b[%dA", rowsDown)
		}
	}
	// Move cursor to correct position (1-based column) using CHA.
	fmt.Fprintf(&out, "\x1b[%dG", cursorCol+1)

	// Show cursor
	out.WriteString("\x1b[?25h")

	_, err := r.out.Write(out.Bytes())
	r.lastWidth = currentWidth
	r.lastRows = max(1, currentRows)
	return err
}

func (r *Renderer) ClearLine() {
	fmt.Fprintf(r.out, "\r\x1b[K")
}

func (r *Renderer) NewLine() {
	fmt.Fprintf(r.out, "\r\n")
	r.lastRows = 0
}

func (r *Renderer) RefreshSearchWithMatches(query string, matches []string, selected int, matchPos int) {
	var out bytes.Buffer

	out.WriteString("\x1b[?25l")

	totalRows := r.lastRows
	if r.lastRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", r.lastRows-1)
	}
	for i := 0; i < max(1, totalRows); i++ {
		out.WriteString("\r\x1b[2K")
		if i < max(1, totalRows)-1 {
			out.WriteString("\x1b[1B")
		}
	}
	if totalRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", totalRows-1)
	}

	total := len(matches)
	var currentLine string
	if total > 0 && selected >= 0 && selected < total {
		currentLine = matches[selected]
	}

	// Header line
	if total > 0 {
		fmt.Fprintf(&out, "\x1b[1G(reverse-i-search)`%s': [%d/%d] ", query, selected+1, total)
	} else {
		fmt.Fprintf(&out, "\x1b[1G(reverse-i-search)`%s': ", query)
	}

	// Current match with highlight
	if currentLine != "" && matchPos >= 0 {
		qLen := len(query)
		before := currentLine[:matchPos]
		match := currentLine[matchPos : matchPos+qLen]
		after := currentLine[matchPos+qLen:]
		out.WriteString(before)
		out.WriteString("\x1b[7m")
		out.WriteString(match)
		out.WriteString("\x1b[0m")
		out.WriteString(after)
	} else {
		out.WriteString(currentLine)
	}
	out.WriteString("\x1b[K")

	// Candidate list below (max 5 visible)
	const maxVisible = 5
	if total > 0 {
		pageStart := 0
		if total > maxVisible {
			pageStart = selected / maxVisible * maxVisible
		}
		pageEnd := min(pageStart+maxVisible, total)

		for idx := pageStart; idx < pageEnd; idx++ {
			out.WriteString("\r\n")
			m := matches[idx]
			if idx == selected {
				out.WriteString("\x1b[36m>\x1b[0m ")
			} else {
				out.WriteString("  ")
			}

			// Highlight match in candidate
			qIdx := strings.Index(m, query)
			if qIdx >= 0 {
				qLen := len(query)
				out.WriteString(m[:qIdx])
				out.WriteString("\x1b[7m")
				out.WriteString(m[qIdx : qIdx+qLen])
				out.WriteString("\x1b[0m")
				out.WriteString(m[qIdx+qLen:])
			} else {
				out.WriteString(m)
			}
			out.WriteString("\x1b[K")
		}
	}

	searchWidth := runewidth.StringWidth("(reverse-i-search)`': ") + runewidth.StringWidth(query) + runewidth.StringWidth(currentLine)
	cursorCol := runewidth.StringWidth("(reverse-i-search)`") + runewidth.StringWidth(query)

	rowsUp := 0
	if total > 0 {
		rowsUp = min(maxVisible, total)
	}
	if rowsUp > 0 {
		fmt.Fprintf(&out, "\x1b[%dA", rowsUp)
	}
	fmt.Fprintf(&out, "\x1b[%dG", cursorCol+1)

	out.WriteString("\x1b[?25h")

	_, _ = r.out.Write(out.Bytes())
	r.lastWidth = runewidth.StringWidth(query) + runewidth.StringWidth(currentLine)
	r.lastRows = max(1, rowsForWidth(searchWidth, r.getTerminalWidth())) + rowsUp
}

func (r *Renderer) RefreshWithCompletion(b *buffer.Buffer, candidates [][]rune, selected int) error {
	currentWidth := b.FullWidth()
	cursorPos := b.DisplayWidth(b.Cursor())

	visualPrompt := stripANSI(r.prompt)
	promptWidth := runewidth.StringWidth(visualPrompt)
	termWidth := r.getTerminalWidth()
	currentRows := rowsForWidth(promptWidth+currentWidth, termWidth)
	cursorRow, cursorCol := cursorPosition(promptWidth+cursorPos, termWidth)

	var out bytes.Buffer

	out.WriteString("\x1b[?25l")

	if r.lastRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", r.lastRows-1)
	}
	for i := 0; i < max(1, r.lastRows); i++ {
		out.WriteString("\r\x1b[2K")
		if i < max(1, r.lastRows)-1 {
			out.WriteString("\x1b[1B")
		}
	}
	if r.lastRows > 1 {
		fmt.Fprintf(&out, "\x1b[%dA", r.lastRows-1)
	}

	fmt.Fprintf(&out, "\x1b[1G%s%s", r.prompt, b.String())
	if currentRows > 1 {
		segments := wrapVisualSegments(r.prompt, promptWidth, b.String(), termWidth)
		if len(segments) > 0 {
			out.WriteString("\r")
			out.WriteString(strings.Join(segments, "\r\n"))
		}
	}
	out.WriteString("\x1b[K")

	completionRows := r.formatCompletionList(&out, candidates, selected, termWidth)
	out.WriteString("\r\n")

	if currentRows > 1 {
		rowsUp := cursorRow
		if rowsUp > 0 {
			fmt.Fprintf(&out, "\x1b[%dB", rowsUp)
		}
	}
	fmt.Fprintf(&out, "\x1b[%dA", completionRows+currentRows)
	fmt.Fprintf(&out, "\x1b[%dG", cursorCol+1)

	out.WriteString("\x1b[?25h")

	_, err := r.out.Write(out.Bytes())
	r.lastWidth = currentWidth
	r.lastRows = currentRows + 1 + completionRows
	return err
}

func (r *Renderer) formatCompletionList(out *bytes.Buffer, candidates [][]rune, selected int, termWidth int) int {
	if len(candidates) == 0 {
		return 0
	}

	maxWidth := 0
	for _, c := range candidates {
		w := runewidth.StringWidth(string(c))
		if w > maxWidth {
			maxWidth = w
		}
	}
	maxWidth += 2

	cols := termWidth / maxWidth
	if cols < 1 {
		cols = 1
	}

	rows := (len(candidates) + cols - 1) / cols

	for row := 0; row < rows; row++ {
		if row > 0 {
			out.WriteString("\r\n")
		}
		for col := 0; col < cols; col++ {
			idx := row + col*rows
			if idx >= len(candidates) {
				break
			}
			candidate := string(candidates[idx])
			if idx == selected {
				fmt.Fprintf(out, "\x1b[7m%-*s\x1b[0m", maxWidth-1, candidate)
			} else {
				fmt.Fprintf(out, "%-*s", maxWidth-1, candidate)
			}
		}
	}

	return rows
}

func (r *Renderer) getTerminalWidth() int {
	if s, ok := r.out.(sizedWriter); ok {
		width, _, err := s.GetSize()
		if err == nil && width > 0 {
			return width
		}
	}
	return 80
}

func rowsForWidth(totalWidth, termWidth int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	if totalWidth <= 0 {
		return 1
	}
	rows := totalWidth / termWidth
	if totalWidth%termWidth != 0 {
		rows++
	}
	if rows == 0 {
		return 1
	}
	return rows
}

func cursorPosition(visualWidth, termWidth int) (row int, col int) {
	if termWidth <= 0 {
		termWidth = 80
	}
	row = visualWidth / termWidth
	col = visualWidth % termWidth
	return row, col
}

func wrapVisualSegments(prompt string, promptWidth int, content string, termWidth int) []string {
	if termWidth <= 0 {
		termWidth = 80
	}
	var segments []string
	current := make([]rune, 0, len([]rune(content)))
	currentWidth := 0
	available := termWidth - promptWidth
	if available <= 0 {
		available = termWidth
	}
	for _, r := range []rune(content) {
		w := runewidth.RuneWidth(r)
		if w == 0 {
			w = 1
		}
		limit := termWidth
		if len(segments) == 0 {
			limit = available
		}
		if currentWidth+w > limit && currentWidth > 0 {
			segments = append(segments, string(current))
			current = current[:0]
			currentWidth = 0
			limit = termWidth
		}
		current = append(current, r)
		currentWidth += w
	}
	segments = append(segments, string(current))
	if len(segments) > 0 {
		segments[0] = prompt + segments[0]
	}
	return segments
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
