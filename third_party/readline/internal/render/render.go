package render

import (
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

	// Basic redraw: carriage return, print prompt + content, clear to EOL
	fmt.Fprintf(r.out, "\r%s%s\x1b[K", r.prompt, b.String())

	// Move cursor to correct position
	targetPos := promptWidth + cursorPos
	if targetPos > 0 {
		fmt.Fprintf(r.out, "\r\x1b[%dC", targetPos)
	} else {
		fmt.Fprintf(r.out, "\r")
	}

	r.lastWidth = currentWidth
	return nil
}

func (r *Renderer) ClearLine() {
	fmt.Fprintf(r.out, "\r\x1b[K")
}

func (r *Renderer) NewLine() {
	fmt.Fprintf(r.out, "\r\n")
}
