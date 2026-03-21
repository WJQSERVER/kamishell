package main

import (
	"kamishell/builtin"
	"kamishell/core"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type KamiCompleter struct {
	env *core.Environment
}

func (c *KamiCompleter) Do(line []rune, pos int) ([][]rune, int) {
	var candidates [][]rune
	lineStr := string(line[:pos])

	lastWord, completionPrefix, rawToken := extractCompletionToken(lineStr)
	seen := make(map[string]struct{})

	// 1. Built-ins
	for name := range builtin.Builtins {
		if strings.HasPrefix(name, lastWord) {
			appendUniqueCandidate(&candidates, seen, completionPrefix+name)
		}
	}

	// 2. Environment (Variables/Functions)
	if c.env != nil {
		for _, key := range c.env.Keys() {
			if strings.HasPrefix(key, lastWord) {
				appendUniqueCandidate(&candidates, seen, completionPrefix+key)
			}
		}
	}

	// 3. File paths
	for _, candidate := range completePaths(rawToken, lastWord) {
		appendUniqueCandidate(&candidates, seen, completionPrefix+candidate)
	}

	return candidates, utf8.RuneCountInString(completionPrefix + lastWord)
}

func extractCompletionToken(line string) (token string, prefix string, raw string) {
	if len(line) == 0 || strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
		return "", "", ""
	}

	inDoubleQuote := false
	start := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inDoubleQuote = !inDoubleQuote
			if inDoubleQuote {
				start = i
			}
		case ' ', '\t', '|', '&', '>', '<':
			if !inDoubleQuote {
				start = i + 1
			}
		}
	}

	segment := line[start:]
	if strings.HasPrefix(segment, `"`) {
		return segment[1:], `"`, segment[1:]
	}
	return segment, "", segment
}

func appendUniqueCandidate(candidates *[][]rune, seen map[string]struct{}, candidate string) {
	if _, ok := seen[candidate]; ok {
		return
	}
	seen[candidate] = struct{}{}
	*candidates = append(*candidates, []rune(candidate))
}

func completePaths(rawToken, token string) []string {
	dirPart := filepath.Dir(token)
	basePart := filepath.Base(token)
	if dirPart == "." && !strings.ContainsAny(token, `/\`) {
		dirPart = "."
		basePart = token
	}

	prefix := ""
	searchDir := dirPart
	if strings.ContainsAny(rawToken, `/\`) {
		lastSlash := strings.LastIndexAny(rawToken, `/\`)
		if lastSlash >= 0 {
			prefix = rawToken[:lastSlash+1]
		}
	}
	if searchDir == "" {
		searchDir = "."
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), basePart) {
			continue
		}
		candidate := prefix + entry.Name()
		if entry.IsDir() {
			candidate += "/"
		}
		matches = append(matches, candidate)
	}
	return matches
}
