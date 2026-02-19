package main

import (
	"fmt"
	"kamishell"
	"os"
	"path/filepath"

	"github.com/chzyer/readline"
)

const PROMPT = "kami> "

func main() {
	env := kamishell.NewEnvironment()

	// Load .kamirc
	loadConfig(env)

	if len(os.Args) > 1 {
		// Script mode
		filename := os.Args[1]
		executeFile(filename, env)
	} else {
		// REPL mode
		startRepl(env)
	}
}

func executeFile(filename string, env *kamishell.Environment) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	runInput(string(content), env, false)
}

func startRepl(env *kamishell.Environment) {
	home, _ := os.UserHomeDir()
	historyFile := filepath.Join(home, ".kami_history")

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          PROMPT,
		HistoryFile:     historyFile,
		AutoComplete:    &KamiCompleter{env: env},
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF or ctrl-c
			break
		}

		if line == "" {
			continue
		}

		runInput(line, env, true)
	}
}

func runInput(input string, env *kamishell.Environment, isRepl bool) {
	l := kamishell.NewLexer(input)
	p := kamishell.NewParser(l)

	program := p.ParseProgram()
	result := kamishell.Eval(program, env)
	if result != nil {
		if result.Type() == kamishell.ERROR_OBJ {
			fmt.Fprintf(os.Stderr, "%s\n", result.Inspect())
		} else if isRepl && result.Type() != kamishell.NULL_OBJ {
			fmt.Println(result.Inspect())
		}
	}
}

func loadConfig(env *kamishell.Environment) {
	configs := []string{
		os.ExpandEnv("$HOME/.kamirc"),
		".kamirc",
	}

	for _, path := range configs {
		if _, err := os.Stat(path); err == nil {
			content, _ := os.ReadFile(path)
			runInput(string(content), env, false)
		}
	}
}
