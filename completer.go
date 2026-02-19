package main

import (
	"os"
	"path/filepath"
	"strings"
	"kamishell/builtin"
)

type KamiCompleter struct {
	env *Environment
}

func (c *KamiCompleter) Do(line []rune, pos int) ([][]rune, int) {
	var candidates [][]rune
	lineStr := string(line[:pos])

	// Get the last word being typed
	lastWord := ""
	if len(lineStr) > 0 && !strings.HasSuffix(lineStr, " ") {
		idx := strings.LastIndexAny(lineStr, " \t|&><")
		if idx != -1 {
			lastWord = lineStr[idx+1:]
		} else {
			lastWord = lineStr
		}
	}

	// 1. Built-ins
	for name := range builtin.Builtins {
		if strings.HasPrefix(name, lastWord) {
			candidates = append(candidates, []rune(name))
		}
	}

	// 2. Environment (Variables/Functions)
	if c.env != nil {
		for _, key := range c.env.Keys() {
			if strings.HasPrefix(key, lastWord) {
				candidates = append(candidates, []rune(key))
			}
		}
	}

	// 3. File paths
	files, _ := filepath.Glob(lastWord + "*")
	for _, f := range files {
		if info, err := os.Stat(f); err == nil && info.IsDir() {
			f += "/"
		}
		candidates = append(candidates, []rune(f))
	}

	return candidates, len([]rune(lastWord))
}
