package readline

import (
	"strings"
	"unicode"
)

type Completer interface {
	Do(line []rune, pos int) (candidates [][]rune, length int)
}

type PrefixCompleter struct {
	Candidates []string
}

func (p *PrefixCompleter) Do(line []rune, pos int) ([][]rune, int) {
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

type TreeNode struct {
	Name     string
	Children []*TreeNode
}

type TreeCompleter struct {
	root *TreeNode
}

func NewTreeCompleter() *TreeCompleter {
	return &TreeCompleter{root: &TreeNode{}}
}

func (tc *TreeCompleter) Add(path ...string) {
	current := tc.root
	for _, name := range path {
		found := false
		for _, child := range current.Children {
			if child.Name == name {
				current = child
				found = true
				break
			}
		}
		if !found {
			newNode := &TreeNode{Name: name}
			current.Children = append(current.Children, newNode)
			current = newNode
		}
	}
}

func (tc *TreeCompleter) Do(line []rune, pos int) ([][]rune, int) {
	words := splitWords(line[:pos])

	if len(words) == 0 {
		return tc.getCandidates(tc.root, ""), 0
	}

	endsWithSpace := pos > 0 && unicode.IsSpace(line[pos-1])
	if endsWithSpace {
		prefix := ""
		replaceLen := 0

		current := tc.root
		for i := 0; i < len(words); i++ {
			found := false
			for _, child := range current.Children {
				if child.Name == words[i] {
					current = child
					found = true
					break
				}
			}
			if !found {
				return nil, replaceLen
			}
		}

		return tc.getCandidates(current, prefix), replaceLen
	}

	prefix := words[len(words)-1]
	replaceLen := len([]rune(prefix))

	current := tc.root
	for i := 0; i < len(words)-1; i++ {
		found := false
		for _, child := range current.Children {
			if child.Name == words[i] {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil, replaceLen
		}
	}

	return tc.getCandidates(current, prefix), replaceLen
}

func (tc *TreeCompleter) getCandidates(node *TreeNode, prefix string) [][]rune {
	var matches [][]rune
	for _, child := range node.Children {
		if strings.HasPrefix(child.Name, prefix) {
			matches = append(matches, []rune(child.Name))
		}
	}
	return matches
}

func splitWords(line []rune) []string {
	var words []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	escape := false

	for _, r := range line {
		if escape {
			current.WriteRune(r)
			escape = false
			continue
		}

		if r == '\\' && !inSingleQuote {
			escape = true
			continue
		}

		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if !inDoubleQuote && !inSingleQuote && unicode.IsSpace(r) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

type FuzzyCompleter struct {
	Candidates []string
}

func NewFuzzyCompleter(candidates ...string) *FuzzyCompleter {
	return &FuzzyCompleter{Candidates: candidates}
}

func (fc *FuzzyCompleter) Do(line []rune, pos int) ([][]rune, int) {
	start := pos
	for start > 0 && line[start-1] != ' ' {
		start--
	}
	query := string(line[start:pos])

	var matches [][]rune
	for _, c := range fc.Candidates {
		if fuzzyMatch(query, c) {
			matches = append(matches, []rune(c))
		}
	}
	return matches, pos - start
}

func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}

	queryRunes := []rune(query)
	targetRunes := []rune(target)

	qi := 0
	for _, r := range targetRunes {
		if qi < len(queryRunes) && unicode.ToLower(queryRunes[qi]) == unicode.ToLower(r) {
			qi++
		}
	}
	return qi == len(queryRunes)
}
