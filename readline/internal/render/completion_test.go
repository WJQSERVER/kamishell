package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/WJQSERVER/readline/internal/buffer"
)

func TestRefreshWithCompletionShowsCandidates(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	r.SetPrompt("> ")

	b := buffer.NewBuffer()
	b.SetContent("foo")

	candidates := [][]rune{[]rune("foobar"), []rune("foobaz"), []rune("fooqux")}
	r.RefreshWithCompletion(b, candidates, 0)

	output := out.String()
	if !strings.Contains(output, "foobar") {
		t.Fatalf("expected output to contain 'foobar', got %q", output)
	}
	if !strings.Contains(output, "foobaz") {
		t.Fatalf("expected output to contain 'foobaz', got %q", output)
	}
	if !strings.Contains(output, "fooqux") {
		t.Fatalf("expected output to contain 'fooqux', got %q", output)
	}
}

func TestRefreshWithCompletionHighlightsSelected(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	r.SetPrompt("> ")

	b := buffer.NewBuffer()
	b.SetContent("foo")

	candidates := [][]rune{[]rune("foobar"), []rune("foobaz")}
	r.RefreshWithCompletion(b, candidates, 1)

	output := out.String()
	if !strings.Contains(output, "\x1b[7m") {
		t.Fatal("expected output to contain reverse video escape sequence for selected item")
	}
}

func TestRefreshWithCompletionEmptyCandidates(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)
	r.SetPrompt("> ")

	b := buffer.NewBuffer()
	b.SetContent("foo")

	candidates := [][]rune{}
	r.RefreshWithCompletion(b, candidates, 0)

	if out.Len() == 0 {
		t.Fatal("expected some output even with empty candidates")
	}
}

func TestFormatCompletionListSingleColumn(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)

	candidates := [][]rune{
		[]rune("alpha"),
		[]rune("beta"),
		[]rune("gamma"),
	}

	rows := r.formatCompletionList(&out, candidates, 0, 10)

	if rows != 3 {
		t.Fatalf("expected 3 rows for narrow terminal, got %d", rows)
	}

	output := out.String()
	if strings.Count(output, "\r\n") != 2 {
		t.Fatalf("expected 2 newlines for 3 rows, got %d", strings.Count(output, "\r\n"))
	}
}

func TestFormatCompletionListMultipleColumns(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)

	candidates := [][]rune{
		[]rune("a"),
		[]rune("b"),
		[]rune("c"),
		[]rune("d"),
		[]rune("e"),
		[]rune("f"),
	}

	rows := r.formatCompletionList(&out, candidates, 0, 50)

	if rows < 1 || rows > 3 {
		t.Fatalf("expected 1-3 rows for wide terminal with short items, got %d", rows)
	}
}

func TestFormatCompletionListHighlight(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)

	candidates := [][]rune{[]rune("foo"), []rune("bar")}

	r.formatCompletionList(&out, candidates, 1, 80)

	output := out.String()
	if !strings.Contains(output, "\x1b[7mbar") {
		t.Fatalf("expected 'bar' to be highlighted, got %q", output)
	}
	if strings.Contains(output, "\x1b[7mfoo") {
		t.Fatalf("expected 'foo' to not be highlighted, got %q", output)
	}
}

func TestFormatCompletionListUnicodeWidth(t *testing.T) {
	var out bytes.Buffer
	r := NewRenderer(&out)

	candidates := [][]rune{
		[]rune("你好"),
		[]rune("世界"),
	}

	rows := r.formatCompletionList(&out, candidates, 0, 20)

	if rows < 1 {
		t.Fatalf("expected at least 1 row, got %d", rows)
	}

	output := out.String()
	if !strings.Contains(output, "你好") || !strings.Contains(output, "世界") {
		t.Fatalf("expected output to contain unicode candidates, got %q", output)
	}
}
