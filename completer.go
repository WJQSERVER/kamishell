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

// completionContext holds parsed information about the current completion situation.
type completionContext struct {
	commandName  string // first token on the line (the command)
	currentToken string // token currently being completed (unquoted)
	prevToken    string // previous token (for flag value detection)
	isFirstWord  bool   // true if completing the command name itself
	rawToken     string // raw token as typed (may include quote prefix)
	rawLength    int    // length of the raw token in runes (including quote prefix)
}

func (c *KamiCompleter) Do(line []rune, pos int) ([][]rune, int) {
	var candidates [][]rune
	lineStr := string(line[:pos])
	seen := make(map[string]struct{})

	ctx := parseCompletionContext(lineStr)

	if ctx.isFirstWord {
		// Command name position: complete builtins + external commands + env vars
		c.completeCommandNames(ctx.currentToken, &candidates, seen, "")
		completeExternalCommands(ctx.currentToken, &candidates, seen)
		if c.env != nil {
			for _, key := range c.env.Keys() {
				if strings.HasPrefix(key, ctx.currentToken) {
					appendUniqueCandidate(&candidates, seen, key)
				}
			}
		}
	} else if strings.HasPrefix(ctx.currentToken, "-") && ctx.commandName != "" {
		// Flag position: complete flags from command metadata
		c.completeFlags(ctx.commandName, ctx.currentToken, &candidates, seen)
	} else if ctx.commandName != "" {
		// Argument position: check for env var completion ($PREFIX)
		if strings.HasPrefix(ctx.currentToken, "$") {
			c.completeEnvVars(ctx.currentToken, &candidates, seen)
		} else {
			// Check if previous token is a flag that takes a value
			completed := c.completeFlagValue(ctx.commandName, ctx.prevToken, ctx.currentToken, &candidates, seen)
			if !completed {
				// Try arg completer, then fall back to file paths
				completed = c.completeArgs(ctx.commandName, ctx.currentToken, &candidates, seen)
			}
			if !completed {
				for _, candidate := range completePaths(ctx.rawToken, ctx.currentToken) {
					appendUniqueCandidate(&candidates, seen, candidate)
				}
			}
		}
	}

	return candidates, ctx.rawLength
}

// completeCommandNames adds matching builtin command names to candidates.
func (c *KamiCompleter) completeCommandNames(prefix string, candidates *[][]rune, seen map[string]struct{}, completionPrefix string) {
	for name := range builtin.Builtins {
		if strings.HasPrefix(name, prefix) {
			appendUniqueCandidate(candidates, seen, completionPrefix+name)
		}
	}
}

// completeFlags adds matching flags for the given command to candidates.
func (c *KamiCompleter) completeFlags(cmdName string, token string, candidates *[][]rune, seen map[string]struct{}) {
	m := builtin.GetMeta(cmdName)
	if m == nil {
		return
	}
	for _, f := range m.Flags {
		if f.Long != "" && f.Long != f.Short {
			longFlag := "--" + f.Long
			if strings.HasPrefix(longFlag, token) {
				appendUniqueCandidate(candidates, seen, longFlag)
			}
		}
		if f.Short != "" {
			shortFlag := "-" + f.Short
			if strings.HasPrefix(shortFlag, token) {
				appendUniqueCandidate(candidates, seen, shortFlag)
			}
		}
	}
}

// completeArgs tries the command's ArgCompleter. Returns true if any candidates were added.
func (c *KamiCompleter) completeArgs(cmdName string, token string, candidates *[][]rune, seen map[string]struct{}) bool {
	m := builtin.GetMeta(cmdName)
	if m == nil || m.Completer == nil {
		return false
	}
	// Count positional arg index by checking how many non-flag tokens precede the current one
	argIndex := 0 // simplified: always pass 0 for now
	completions := m.Completer(cmdName, argIndex, token)
	for _, comp := range completions {
		if strings.HasPrefix(comp, token) {
			appendUniqueCandidate(candidates, seen, comp)
		}
	}
	return len(completions) > 0
}

// completeFlagValue checks if prevToken is a flag that takes a value, and if so,
// uses the flag's ValueCompleter to generate candidates. Returns true if candidates were added.
func (c *KamiCompleter) completeFlagValue(cmdName string, prevToken string, token string, candidates *[][]rune, seen map[string]struct{}) bool {
	if prevToken == "" || !strings.HasPrefix(prevToken, "-") {
		return false
	}
	m := builtin.GetMeta(cmdName)
	if m == nil {
		return false
	}
	f := m.FindFlagByToken(prevToken)
	if f == nil || f.Type == builtin.FlagBool || f.ValueCompleter == nil {
		return false
	}
	completions := f.ValueCompleter(cmdName, 0, token)
	for _, comp := range completions {
		if strings.HasPrefix(comp, token) {
			appendUniqueCandidate(candidates, seen, comp)
		}
	}
	return len(completions) > 0
}

// completeEnvVars completes environment variable names with $ prefix.
func (c *KamiCompleter) completeEnvVars(token string, candidates *[][]rune, seen map[string]struct{}) {
	// Strip the $ prefix for matching
	prefix := strings.TrimPrefix(token, "$")

	// Complete from OS environment
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		name := parts[0]
		if strings.HasPrefix(name, prefix) {
			appendUniqueCandidate(candidates, seen, "$"+name)
		}
	}

	// Complete from script environment
	if c.env != nil {
		for _, key := range c.env.Keys() {
			if strings.HasPrefix(key, prefix) {
				appendUniqueCandidate(candidates, seen, "$"+key)
			}
		}
	}
}

// completeExternalCommands scans PATH for executable files matching the prefix.
func completeExternalCommands(prefix string, candidates *[][]rune, seen map[string]struct{}) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return
	}

	duplicateCheck := make(map[string]bool)
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if duplicateCheck[name] {
				continue
			}
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 && !info.IsDir() {
				duplicateCheck[name] = true
				appendUniqueCandidate(candidates, seen, name)
			}
		}
	}
}

// parseCompletionContext analyzes the input line to determine what kind of completion is needed.
func parseCompletionContext(line string) completionContext {
	ctx := completionContext{}

	if len(line) == 0 {
		ctx.isFirstWord = true
		return ctx
	}

	// Check if we're starting a new token (line ends with space)
	endsWithSpace := line[len(line)-1] == ' ' || line[len(line)-1] == '\t'

	// Parse tokens from the beginning to find the command name
	tokens := tokenizeForCompletion(line)
	currentToken := ""

	if endsWithSpace {
		currentToken = ""
	} else if len(tokens) > 0 {
		currentToken = tokens[len(tokens)-1]
	}

	// Determine if we're completing the first word
	if len(tokens) == 0 || (endsWithSpace && len(tokens) == 0) {
		ctx.isFirstWord = true
	} else if !endsWithSpace && len(tokens) == 1 {
		ctx.isFirstWord = true
	}

	// The command name is the first non-flag token
	ctx.commandName = ""
	for _, t := range tokens {
		if !strings.HasPrefix(t, "-") {
			ctx.commandName = t
			break
		}
	}

	ctx.currentToken = currentToken
	ctx.rawToken = currentToken

	// Set prevToken for flag value detection
	if !endsWithSpace && len(tokens) >= 2 {
		ctx.prevToken = tokens[len(tokens)-2]
	} else if endsWithSpace && len(tokens) >= 1 {
		ctx.prevToken = tokens[len(tokens)-1]
	}

	// Calculate raw length including any quote prefix
	ctx.rawLength = utf8.RuneCountInString(currentToken)
	if !endsWithSpace && len(line) > 0 {
		// Find the start of the current token in the raw line
		// Look backwards for the last unquoted space or start of line
		tokenStart := findTokenStart(line)
		if tokenStart < len(line) && line[tokenStart] == '"' {
			ctx.rawLength = utf8.RuneCountInString(line[tokenStart:])
			ctx.rawToken = line[tokenStart:]
		}
	}

	return ctx
}

// findTokenStart finds the start position of the current token in the line.
func findTokenStart(line string) int {
	inQuote := false
	start := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case ' ', '\t', '|', '&', '>', '<':
			if !inQuote {
				start = i + 1
			}
		}
	}
	return start
}

// tokenizeForCompletion splits a line into tokens, respecting double quotes and shell metacharacters.
func tokenizeForCompletion(line string) []string {
	var tokens []string
	var current strings.Builder
	inDoubleQuote := false
	escaped := false
	hasContent := false

	for i := 0; i < len(line); i++ {
		b := line[i]
		if escaped {
			current.WriteByte(b)
			escaped = false
			hasContent = true
			continue
		}
		switch b {
		case '\\':
			if inDoubleQuote {
				escaped = true
			} else {
				current.WriteByte(b)
				hasContent = true
			}
		case '"':
			inDoubleQuote = !inDoubleQuote
			hasContent = true
		case ' ', '\t':
			if inDoubleQuote {
				current.WriteByte(b)
				hasContent = true
			} else if hasContent {
				tokens = append(tokens, current.String())
				current.Reset()
				hasContent = false
			}
		case '|', '&', '>', '<':
			if !inDoubleQuote {
				if hasContent {
					tokens = append(tokens, current.String())
					current.Reset()
					hasContent = false
				}
				// Skip shell metacharacters - they start a new command context
				tokens = nil
			} else {
				current.WriteByte(b)
				hasContent = true
			}
		default:
			current.WriteByte(b)
			hasContent = true
		}
	}

	if hasContent {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func appendUniqueCandidate(candidates *[][]rune, seen map[string]struct{}, candidate string) {
	if _, ok := seen[candidate]; ok {
		return
	}
	seen[candidate] = struct{}{}
	*candidates = append(*candidates, []rune(candidate))
}

func completePaths(rawToken, token string) []string {
	// Handle tilde expansion
	tildePrefix := ""
	if strings.HasPrefix(token, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		tildePrefix = "~/"
		token = strings.TrimPrefix(token, "~")
		if strings.HasPrefix(token, "/") {
			token = token[1:]
		}
		// Replace ~ with home dir for filesystem operations
		rawToken = strings.TrimPrefix(rawToken, "~")
		if strings.HasPrefix(rawToken, "/") {
			rawToken = rawToken[1:]
		}
		// Prepend home to token for directory traversal
		token = filepath.Join(home, token)
	}

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
		candidate := tildePrefix + prefix + entry.Name()
		if entry.IsDir() {
			candidate += "/"
		}
		matches = append(matches, candidate)
	}
	return matches
}
