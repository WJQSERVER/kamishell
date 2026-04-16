package readline

import (
	"testing"
)

func TestTreeCompleterBasic(t *testing.T) {
	tc := NewTreeCompleter()
	tc.Add("git", "status")
	tc.Add("git", "commit")
	tc.Add("git", "push")
	tc.Add("npm", "install")
	tc.Add("npm", "run")

	candidates, length := tc.Do([]rune("git "), 4)
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates for 'git ', got %d", len(candidates))
	}
	if length != 0 {
		t.Fatalf("expected length 0, got %d", length)
	}
}

func TestTreeCompleterWithPrefix(t *testing.T) {
	tc := NewTreeCompleter()
	tc.Add("git", "status")
	tc.Add("git", "commit")
	tc.Add("git", "push")

	candidates, length := tc.Do([]rune("git st"), 6)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for 'git st', got %d", len(candidates))
	}
	if string(candidates[0]) != "status" {
		t.Fatalf("expected 'status', got %q", string(candidates[0]))
	}
	if length != 2 {
		t.Fatalf("expected length 2, got %d", length)
	}
}

func TestTreeCompleterRootLevel(t *testing.T) {
	tc := NewTreeCompleter()
	tc.Add("git", "status")
	tc.Add("npm", "install")

	candidates, _ := tc.Do([]rune("gi"), 2)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for 'gi', got %d", len(candidates))
	}
	if string(candidates[0]) != "git" {
		t.Fatalf("expected 'git', got %q", string(candidates[0]))
	}
}

func TestTreeCompleterNoMatch(t *testing.T) {
	tc := NewTreeCompleter()
	tc.Add("git", "status")

	candidates, _ := tc.Do([]rune("svn "), 4)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for 'svn ', got %d", len(candidates))
	}
}

func TestTreeCompleterDeepPath(t *testing.T) {
	tc := NewTreeCompleter()
	tc.Add("docker", "container", "ls")
	tc.Add("docker", "container", "rm")
	tc.Add("docker", "image", "ls")

	candidates, _ := tc.Do([]rune("docker container "), 17)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates for 'docker container ', got %d: %v", len(candidates), candidates)
	}
}

func TestFuzzyCompleterBasic(t *testing.T) {
	fc := NewFuzzyCompleter("foobar", "foobaz", "barfoo")

	candidates, length := fc.Do([]rune("fb"), 2)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates for 'fb', got %d", len(candidates))
	}
	if length != 2 {
		t.Fatalf("expected length 2, got %d", length)
	}
}

func TestFuzzyCompleterCaseInsensitive(t *testing.T) {
	fc := NewFuzzyCompleter("Foobar", "FOOBAZ")

	candidates, _ := fc.Do([]rune("fb"), 2)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates for 'fb' (case insensitive), got %d", len(candidates))
	}
}

func TestFuzzyCompleterExactMatch(t *testing.T) {
	fc := NewFuzzyCompleter("foobar", "foobaz")

	candidates, _ := fc.Do([]rune("foobar"), 6)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for exact match 'foobar', got %d", len(candidates))
	}
	if string(candidates[0]) != "foobar" {
		t.Fatalf("expected 'foobar', got %q", string(candidates[0]))
	}
}

func TestFuzzyCompleterNoMatch(t *testing.T) {
	fc := NewFuzzyCompleter("foobar")

	candidates, _ := fc.Do([]rune("xyz"), 3)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for 'xyz', got %d", len(candidates))
	}
}

func TestFuzzyCompleterEmptyQuery(t *testing.T) {
	fc := NewFuzzyCompleter("foo", "bar")

	candidates, _ := fc.Do([]rune(""), 0)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates for empty query, got %d", len(candidates))
	}
}

func TestFuzzyCompleterUnicode(t *testing.T) {
	fc := NewFuzzyCompleter("你好世界", "你好中国")

	candidates, _ := fc.Do([]rune("你世"), 2)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for '你世', got %d", len(candidates))
	}
	if string(candidates[0]) != "你好世界" {
		t.Fatalf("expected '你好世界', got %q", string(candidates[0]))
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"git status", []string{"git", "status"}},
		{"  multiple   spaces  ", []string{"multiple", "spaces"}},
		{`"quoted string"`, []string{"quoted string"}},
		{`git "commit message"`, []string{"git", "commit message"}},
	}

	for _, tt := range tests {
		result := splitWords([]rune(tt.input))
		if len(result) != len(tt.expected) {
			t.Fatalf("splitWords(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
		for i, w := range result {
			if w != tt.expected[i] {
				t.Fatalf("splitWords(%q)[%d] = %q, expected %q", tt.input, i, w, tt.expected[i])
			}
		}
	}
}
