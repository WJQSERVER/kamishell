package main

import (
	"flag"
	"fmt"
	"kamishell"
	"os"
	"path/filepath"

	chreadline "github.com/chzyer/readline"
	wjqreadline "github.com/WJQSERVER/readline"
)

const PROMPT = "kami> "

var (
	readlineLib = flag.String("readline", "chzyer", "Select readline library: chzyer (default) or wjq (experimental)")
)

func main() {
	flag.Parse()
	env := kamishell.NewEnvironment()

	// Load .kamirc
	loadConfig(env)

	args := flag.Args()
	if len(args) > 0 {
		// Script mode
		filename := args[0]
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

	if *readlineLib == "wjq" {
		startWjqRepl(env, historyFile)
	} else {
		startChzyerRepl(env, historyFile)
	}
}

func startChzyerRepl(env *kamishell.Environment, historyFile string) {
	rl, err := chreadline.NewEx(&chreadline.Config{
		Prompt:          PROMPT,
		HistoryFile:     historyFile,
		AutoComplete:    &KamiCompleter{env: env},
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing chzyer readline: %v\n", err)
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

func startWjqRepl(env *kamishell.Environment, historyFile string) {
	// WJQSERVER/readline implementation
	// Note: WJQSERVER/readline might have different ways to handle history files
	cfg := &wjqreadline.Config{
		Prompt:    PROMPT,
		Completer: &KamiCompleter{env: env},
	}

	rl, err := wjqreadline.NewInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing wjq readline: %v\n", err)
		return
	}
	// For now, history in wjq is in-memory or needs more setup if we want it persistent
	// as I saw "History interface" and "NewHistory()" in go doc.

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == wjqreadline.ErrInterrupt {
				fmt.Println("^C")
				continue
			}
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
