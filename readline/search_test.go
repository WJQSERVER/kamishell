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
	rl.rebuildSearchMatches()
	rl.mu.Unlock()

	if rl.searchResult != "" {
		t.Fatalf("expected empty result with empty history, got %q", rl.searchResult)
	}
}

func TestSearchMatchPosition(t *testing.T) {
	h := NewHistory()
	h.Append("git status")
	h.Append("echo hello world")
	h.Append("git push origin main")

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
	rl.searchQuery.SetContent("origin")
	rl.rebuildSearchMatches()
	rl.mu.Unlock()

	if len(rl.searchMatches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(rl.searchMatches))
	}
	if rl.searchMatches[0].matchPos != 9 {
		t.Fatalf("expected matchPos=9, got %d", rl.searchMatches[0].matchPos)
	}
}

func TestSearchMultipleMatches(t *testing.T) {
	h := NewHistory()
	h.Append("go build")
	h.Append("go test")
	h.Append("go vet")
	h.Append("echo done")

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
	rl.searchQuery.SetContent("go")
	rl.rebuildSearchMatches()
	rl.mu.Unlock()

	if len(rl.searchMatches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(rl.searchMatches))
	}

	// Most recent should be first (go vet)
	if rl.searchResult != "go vet" {
		t.Fatalf("expected first match 'go vet', got %q", rl.searchResult)
	}

	// Navigate to next
	rl.mu.Lock()
	rl.searchPrevMatch()
	rl.mu.Unlock()

	if rl.searchResult != "go test" {
		t.Fatalf("expected second match 'go test', got %q", rl.searchResult)
	}

	// Navigate to next
	rl.mu.Lock()
	rl.searchPrevMatch()
	rl.mu.Unlock()

	if rl.searchResult != "go build" {
		t.Fatalf("expected third match 'go build', got %q", rl.searchResult)
	}

	// Wrap around
	rl.mu.Lock()
	rl.searchPrevMatch()
	rl.mu.Unlock()

	if rl.searchResult != "go vet" {
		t.Fatalf("expected wrap to 'go vet', got %q", rl.searchResult)
	}
}
