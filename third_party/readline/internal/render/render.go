package render

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mattn/go-runewidth"
	"github.com/WJQSERVER/readline/internal/buffer"
)

type Renderer struct {
	out          io.Writer
	prompt       string
	lastWidth    int
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

	var out bytes.Buffer

	// Hide cursor to prevent jitter
	out.WriteString("\x1b[?25l")

	// Move to column 1 and clear to EOL, then print prompt and content
	fmt.Fprintf(&out, "\x1b[1G%s%s\x1b[K", r.prompt, b.String())

	// Move cursor to correct position (1-based column) using CHA
	fmt.Fprintf(&out, "\x1b[%dG", promptWidth+cursorPos+1)

	// Show cursor
	out.WriteString("\x1b[?25h")

	_, err := r.out.Write(out.Bytes())
	r.lastWidth = currentWidth
	return err
}

func (r *Renderer) ClearLine() {
	fmt.Fprintf(r.out, "\r\x1b[K")
}

func (r *Renderer) NewLine() {
	fmt.Fprintf(r.out, "\r\n")
}
