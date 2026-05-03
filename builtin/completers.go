package builtin

import (
	"os"
	"path/filepath"
	"strings"
)

// completeBuiltinNames returns builtin command names matching the prefix.
func completeBuiltinNames(cmdName string, argIndex int, prefix string) []string {
	var result []string
	for name := range Builtins {
		if strings.HasPrefix(name, prefix) {
			result = append(result, name)
		}
	}
	return result
}

// completeCommandNames returns builtin command names + external commands from PATH.
func completeCommandNames(cmdName string, argIndex int, prefix string) []string {
	var result []string

	// Builtin commands
	for name := range Builtins {
		if strings.HasPrefix(name, prefix) {
			result = append(result, name)
		}
	}

	// External commands from PATH
	result = append(result, completeExternalCommands(prefix)...)

	return result
}

// completeDirectoryPaths returns only directory paths matching the prefix.
func completeDirectoryPaths(cmdName string, argIndex int, prefix string) []string {
	dir, base := splitPath(prefix)
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		candidate := dir + name + "/"
		result = append(result, candidate)
	}
	return result
}

// completeExternalCommands scans PATH for executable files matching the prefix.
func completeExternalCommands(prefix string) []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if seen[name] {
				continue
			}
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			// Check if executable
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 && !info.IsDir() {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// splitPath splits a path into directory and base components.
func splitPath(path string) (dir, base string) {
	if path == "" {
		return "", ""
	}
	// Find last separator
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			lastSlash = i
			break
		}
	}
	if lastSlash < 0 {
		return "", path
	}
	return path[:lastSlash+1], path[lastSlash+1:]
}

// completeEnvVarNames returns environment variable names matching the prefix.
func completeEnvVarNames(cmdName string, argIndex int, prefix string) []string {
	var result []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		name := parts[0]
		if strings.HasPrefix(name, prefix) {
			result = append(result, name)
		}
	}
	return result
}
