package readline

import (
	"os"
	"testing"

	"github.com/WJQSERVER/readline/internal/input"
)

type testCompleter struct {
	candidates [][]rune
	length     int
}

func (tc *testCompleter) Do(line []rune, pos int) ([][]rune, int) {
	return tc.candidates, tc.length
}

func TestCompletionModeEnterMultipleCandidates(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if !rl.completionMode {
		t.Fatal("expected completionMode to be true")
	}
	if len(rl.completionCandidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(rl.completionCandidates))
	}
	if rl.completionSelected != 0 {
		t.Fatalf("expected selected index 0, got %d", rl.completionSelected)
	}
}

func TestCompletionModeTabCycles(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected initial selection 0, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 1 {
		t.Fatalf("expected selection 1 after Tab, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 2 {
		t.Fatalf("expected selection 2 after second Tab, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyTab})
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected selection 0 after wrap, got %d", rl.completionSelected)
	}
}

func TestCompletionModeShiftTabReverseCycles(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar"), []rune("baz")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionSelected != 0 {
		t.Fatalf("expected initial selection 0, got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyShiftTab})
	rl.mu.Unlock()

	if rl.completionSelected != 2 {
		t.Fatalf("expected selection 2 after Shift+Tab (wrap to last), got %d", rl.completionSelected)
	}

	rl.mu.Lock()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyShiftTab})
	rl.mu.Unlock()

	if rl.completionSelected != 1 {
		t.Fatalf("expected selection 1 after second Shift+Tab, got %d", rl.completionSelected)
	}
}

func TestCompletionModeSingleCandidateAppliesDirectly(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false for single candidate")
	}
	if rl.buffer.String() != "foobar" {
		t.Fatalf("expected buffer 'foobar', got %q", rl.buffer.String())
	}
}

func TestCompletionModeCancel(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.cancelCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after cancel")
	}
	if rl.completionCandidates != nil {
		t.Fatal("expected completionCandidates to be nil after cancel")
	}
	if rl.completionReplaceLen != 0 {
		t.Fatal("expected completionReplaceLen to be 0 after cancel")
	}
}

func TestCompletionModeApply(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar"), []rune("foobaz")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.completionSelected = 1
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.buffer.String() != "foobaz" {
		t.Fatalf("expected buffer 'foobaz', got %q", rl.buffer.String())
	}
	if rl.completionMode {
		t.Fatal("expected completionMode to be false after apply")
	}
	if rl.completionReplaceLen != 0 {
		t.Fatal("expected completionReplaceLen to be 0 after apply")
	}
}

func TestCompletionModeNoCandidates(t *testing.T) {
	tc := &testCompleter{
		candidates: nil,
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false with no candidates")
	}
}

func TestCompletionModeNoCompleter(t *testing.T) {
	cfg := &Config{
		Prompt:  "> ",
		History: NewHistory(),
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false with no completer")
	}
}

func TestCompletionModeApplyTwice(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.applyCompletion()
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.buffer.String() != "foo" {
		t.Fatalf("expected buffer 'foo', got %q", rl.buffer.String())
	}
}

func TestCompletionModeApplyWithInvalidIndex(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.completionSelected = 99
	rl.applyCompletion()
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after apply with invalid index")
	}
}

func TestCompletionModeEnterApplies(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foobar"), []rune("foobaz")},
		length:     3,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.buffer.SetContent("foo")
	rl.handleCompletion()
	rl.completionSelected = 1
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyEnter})
	rl.mu.Unlock()

	if rl.buffer.String() != "foobaz" {
		t.Fatalf("expected buffer 'foobaz' after Enter, got %q", rl.buffer.String())
	}
}

func TestCompletionModeEscCancels(t *testing.T) {
	tc := &testCompleter{
		candidates: [][]rune{[]rune("foo"), []rune("bar")},
		length:     0,
	}

	cfg := &Config{
		Prompt:    "> ",
		Completer: tc,
		History:   NewHistory(),
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.handleCompletion()
	rl.handleCompletionMode(input.InputEvent{Key: input.KeyEsc})
	rl.mu.Unlock()

	if rl.completionMode {
		t.Fatal("expected completionMode to be false after Esc")
	}
}

func TestSplitWordsBasic(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"  hello   world  ", []string{"hello", "world"}},
		{"single", []string{"single"}},
		{"", []string(nil)},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsDoubleQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`"hello world"`, []string{"hello world"}},
		{`echo "hello world"`, []string{"echo", "hello world"}},
		{`cmd "arg with spaces" more`, []string{"cmd", "arg with spaces", "more"}},
		{`"nested" "quotes"`, []string{"nested", "quotes"}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsEscapedQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`"arg with \" quote"`, []string{`arg with " quote`}},
		{`cmd "arg with \" quote"`, []string{"cmd", `arg with " quote`}},
		{`"escaped \" and \" quotes"`, []string{`escaped " and " quotes`}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsSingleQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`'hello world'`, []string{"hello world"}},
		{`echo 'hello world'`, []string{"echo", "hello world"}},
		{`cmd 'arg with spaces' more`, []string{"cmd", "arg with spaces", "more"}},
		{`'no escape \n here'`, []string{`no escape \n here`}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsBackslashEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`hello\ world`, []string{"hello world"}},
		{`echo hello\ world`, []string{"echo", "hello world"}},
		{`path\ with\ spaces`, []string{"path with spaces"}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsMixed(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`cmd "double" 'single' plain`, []string{"cmd", "double", "single", "plain"}},
		{`cmd "arg 'with' quotes"`, []string{"cmd", "arg 'with' quotes"}},
		{`cmd 'arg "with" quotes'`, []string{"cmd", `arg "with" quotes`}},
		{`cmd "escaped \" inside" 'literal \n'`, []string{"cmd", `escaped " inside`, `literal \n`}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsEmptyQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`""`, []string{""}},
		{`''`, []string{""}},
		{`cmd ""`, []string{"cmd", ""}},
		{`cmd ''`, []string{"cmd", ""}},
		{`cmd "" arg2`, []string{"cmd", "", "arg2"}},
		{`cmd '' arg2`, []string{"cmd", "", "arg2"}},
		{`"" ""`, []string{"", ""}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v (len=%d), want %v (len=%d)", tt.input, result, len(result), tt.expected, len(tt.expected))
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsTrailingBackslash(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`cmd\`, []string{`cmd\`}},
		{`echo test\`, []string{"echo", `test\`}},
		{`cmd arg\`, []string{"cmd", `arg\`}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v (len=%d), want %v (len=%d)", tt.input, result, len(result), tt.expected, len(tt.expected))
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}

func TestSplitWordsPOSIXDoubleQuoteEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`"a\b"`, []string{`a\b`}},
		{`"a\\b"`, []string{`a\b`}},
		{`"a\"b"`, []string{`a"b`}},
		{`"a\$b"`, []string{`a$b`}},
		{`"a\` + "`" + `b"`, []string{"a`b"}},
		{`"\n"`, []string{`\n`}},
		{`"a\` + "\n" + `b"`, []string{`ab`}},
		{`"test\` + "\n" + `value"`, []string{`testvalue`}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Errorf("splitWords(%q) = %v (len=%d), want %v (len=%d)", tt.input, result, len(result), tt.expected, len(tt.expected))
			continue
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Errorf("splitWords(%q)[%d] = %q, want %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}
