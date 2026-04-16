package readline

import (
	"os"
	"testing"
)

func TestReverseSearchFindsMatch(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("git commit")
	h.Append("git push")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("push")
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "git push" {
		t.Fatalf("expected 'git push', got %q", rl.searchResult)
	}
}

func TestReverseSearchFindsMostRecentMatch(t *testing.T) {
	h := NewHistory()
	h.Append("echo first")
	h.Append("echo second")
	h.Append("echo third")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("echo")
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "echo third" {
		t.Fatalf("expected most recent 'echo third', got %q", rl.searchResult)
	}
}

func TestReverseSearchPrevMatch(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("git commit")
	h.Append("git push")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("git")
	rl.updateSearchResult()

	if rl.searchResult != "git push" {
		t.Fatalf("expected first match 'git push', got %q", rl.searchResult)
	}

	rl.searchPrevMatch()
	if rl.searchResult != "git commit" {
		t.Fatalf("expected prev match 'git commit', got %q", rl.searchResult)
	}

	rl.searchPrevMatch()
	if rl.searchResult != "git status" {
		t.Fatalf("expected prev match 'git status', got %q", rl.searchResult)
	}
}

func TestReverseSearchNextMatch(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("git commit")
	h.Append("git push")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("git")
	rl.updateSearchResult()

	rl.searchPrevMatch()
	rl.searchPrevMatch()

	if rl.searchResult != "git status" {
		t.Fatalf("expected 'git status', got %q", rl.searchResult)
	}

	rl.searchNextMatch()
	if rl.searchResult != "git commit" {
		t.Fatalf("expected next match 'git commit', got %q", rl.searchResult)
	}

	rl.searchNextMatch()
	if rl.searchResult != "git push" {
		t.Fatalf("expected next match 'git push', got %q", rl.searchResult)
	}
}

func TestReverseSearchNoMatch(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("git commit")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("nonexistent")
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "" {
		t.Fatalf("expected empty result for no match, got %q", rl.searchResult)
	}
}

func TestReverseSearchEmptyQuery(t *testing.T) {
	h := NewHistory()
	h.Append("git status")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "" {
		t.Fatalf("expected empty result for empty query, got %q", rl.searchResult)
	}
}

func TestReverseSearchBackspace(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("git commit")
	h.Append("git push")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("push")
	rl.updateSearchResult()

	if rl.searchResult != "git push" {
		t.Fatalf("expected 'git push', got %q", rl.searchResult)
	}

	rl.searchQuery.SetContent("pu")
	rl.updateSearchResult()

	if rl.searchResult != "git push" {
		t.Fatalf("expected 'git push' after backspace, got %q", rl.searchResult)
	}

	rl.searchQuery.SetContent("")
	rl.updateSearchResult()

	if rl.searchResult != "" {
		t.Fatalf("expected empty result after clearing query, got %q", rl.searchResult)
	}
}

func TestReverseSearchUnicode(t *testing.T) {
	h := NewHistory()
	h.Append("你好世界")
	h.Append("こんにちは")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("你好")
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "你好世界" {
		t.Fatalf("expected '你好世界', got %q", rl.searchResult)
	}
}

func TestReverseSearchPrevMatchStopsAtOldest(t *testing.T) {
	h := NewHistory()
	h.Append("first")
	h.Append("second")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("first")
	rl.updateSearchResult()
	rl.searchPrevMatch()
	rl.mu.Unlock()

	if rl.searchResult != "first" {
		t.Fatalf("expected 'first' (oldest), got %q", rl.searchResult)
	}
}

func TestReverseSearchNextMatchStopsAtNewest(t *testing.T) {
	h := NewHistory()
	h.Append("first")
	h.Append("second")

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("second")
	rl.updateSearchResult()
	rl.searchNextMatch()
	rl.mu.Unlock()

	if rl.searchResult != "second" {
		t.Fatalf("expected 'second' (newest), got %q", rl.searchResult)
	}
}

func TestReverseSearchWithEmptyHistory(t *testing.T) {
	h := NewHistory()

	cfg := &Config{
		Prompt:  "> ",
		History: h,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	cfg.Init()

	rl, err := NewInstance(cfg)
	if err != nil {
		t.Fatalf("NewInstance failed: %v", err)
	}

	rl.mu.Lock()
	rl.startSearch()
	rl.searchQuery.SetContent("anything")
	rl.updateSearchResult()
	rl.mu.Unlock()

	if rl.searchResult != "" {
		t.Fatalf("expected empty result with empty history, got %q", rl.searchResult)
	}
}
