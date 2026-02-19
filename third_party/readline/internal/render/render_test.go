package render

import (
	"bytes"
	"strings"
	"testing"
	"github.com/WJQSERVER/readline/internal/buffer"
)

func TestRenderer_Refresh(t *testing.T) {
	var buf bytes.Buffer
	r := NewRenderer(&buf)
	r.SetPrompt("\x1b[36mkami>\x1b[0m ") // Visual width 6

	b := buffer.NewBuffer()
	b.Insert('1')
	b.Insert('1')
	b.Insert('1') // Visual width 3

	r.Refresh(b)

	output := buf.String()
	// Output should contain "\x1b[10G" at the end to move cursor to column 10 (6+3+1)
	if !strings.Contains(output, "\x1b[10G") {
		t.Errorf("Expected output to contain cursor move to 10G, got %q", output)
	}
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"\x1b[36mkami>\x1b[0m ", "kami> "},
		{"plain text", "plain text"},
		{"\x1b[1;31mBold Red\x1b[0m", "Bold Red"},
	}

	for _, c := range cases {
		got := stripANSI(c.input)
		if got != c.expected {
			t.Errorf("stripANSI(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}
