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

func (r *Renderer) RefreshSearch(query *buffer.Buffer, result string) {
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

	fmt.Fprintf(&out, "\x1b[1G(reverse-i-search)`%s': %s", query.String(), result)
	out.WriteString("\x1b[K")

	promptLen := runewidth.StringWidth("(reverse-i-search)`': ") + query.FullWidth()
	fmt.Fprintf(&out, "\x1b[%dG", promptLen+1)

	out.WriteString("\x1b[?25h")

	r.out.Write(out.Bytes())
	r.lastWidth = query.FullWidth() + runewidth.StringWidth(result)
	r.lastRows = 1
}

func (r *Renderer) RefreshWithCompletion(b *buffer.Buffer, candidates [][]rune, selected int) error {
	currentWidth := b.FullWidth()
	cursorPos := b.DisplayWidth(b.Cursor())

	visualPrompt := stripANSI(r.prompt)
	promptWidth := runewidth.StringWidth(visualPrompt)
	termWidth := r.getTerminalWidth()

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
	out.WriteString("\x1b[K")

	completionRows := r.formatCompletionList(&out, candidates, selected, termWidth)
	out.WriteString("\r\n")

	cursorCol := promptWidth + cursorPos
	fmt.Fprintf(&out, "\x1b[%dA", completionRows+1)
	fmt.Fprintf(&out, "\x1b[%dG", cursorCol+1)

	out.WriteString("\x1b[?25h")

	_, err := r.out.Write(out.Bytes())
	r.lastWidth = currentWidth
	r.lastRows = 1 + completionRows
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
