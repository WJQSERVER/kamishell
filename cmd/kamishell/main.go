package main

import (
	"flag"
	"fmt"
	"kamishell"
	"os"
	"path/filepath"
	"strings"

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
	// Cyan prompt for WJQ version to distinguish it
	cyanPrompt := "\033[36mkami>\033[0m "

	cfg := &wjqreadline.Config{
		Prompt:    cyanPrompt,
		Completer: &KamiCompleter{env: env},
		History:   NewWjqFileHistory(historyFile),
	}

	rl, err := wjqreadline.NewInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing wjq readline: %v\n", err)
		return
	}

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == wjqreadline.ErrInterrupt {
				fmt.Println("^C")
				continue
			}
			// EOF or other fatal errors
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

type WjqFileHistory struct {
	wjqreadline.History
	filepath string
}

func NewWjqFileHistory(path string) *WjqFileHistory {
	h := &WjqFileHistory{
		History:  wjqreadline.NewHistory(),
		filepath: path,
	}
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				h.History.Append(line)
			}
		}
	}
	return h
}

func (h *WjqFileHistory) Append(line string) {
	h.History.Append(line)
	f, err := os.OpenFile(h.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(line + "\n")
	}
}
