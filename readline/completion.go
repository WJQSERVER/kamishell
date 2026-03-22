package readline

import "strings"

type Completer interface {
	Do(line []rune, pos int) (candidates [][]rune, length int)
}

type PrefixCompleter struct {
	Candidates []string
}

func (p *PrefixCompleter) Do(line []rune, pos int) ([][]rune, int) {
	// Simple implementation: complete the word before cursor
	start := pos
	for start > 0 && line[start-1] != ' ' {
		start--
	}
	prefix := string(line[start:pos])

	var matches [][]rune
	for _, c := range p.Candidates {
		if strings.HasPrefix(c, prefix) {
			matches = append(matches, []rune(c))
		}
	}
	return matches, pos - start
}
