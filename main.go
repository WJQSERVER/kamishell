package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wjqreadline "github.com/WJQSERVER/readline"
	chreadline "github.com/chzyer/readline"
	"kamishell/builtin"
	"kamishell/core"
	"kamishell/make"
)

func init() {
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name:        "make",
		Description: "Build system inspired by CMake using .km scripts",
		Action:      make.Make,
	})
}

var (
	readlineLib = flag.String("readline", "wjq", "Select readline library: wjq (default) or base (legacy)")
)

func main() {
	flag.Parse()
	env := core.NewEnvironment()

	// Load .kamirc
	loadConfig(env)

	args := flag.Args()
	if len(args) > 0 {
		if shouldRunAsBuiltin(args[0]) {
			runBuiltinArgs(args, env)
			return
		}

		// Script mode
		filename := args[0]
		executeFile(filename, env)
	} else {
		// REPL mode
		startRepl(env)
	}
}

func shouldRunAsBuiltin(name string) bool {
	info, err := os.Stat(name)
	isDir := err == nil && info.IsDir()
	isFile := err == nil && !isDir
	if isFile {
		return false
	}
	_, ok := builtin.Builtins[name]
	return ok
}

func runBuiltinArgs(args []string, env *core.Environment) {
	if len(args) == 0 {
		return
	}

	cmd, ok := builtin.Builtins[args[0]]
	if !ok {
		runInput(strings.Join(args, " "), env, false)
		return
	}

	exitCode := cmd.Action(args[1:], env, os.Stdin, os.Stdout, os.Stderr)
	if exitCode != 0 {
		fmt.Fprintf(os.Stderr, "ERROR (%s): builtin %s failed (code: %d)\n", cmd.Name, cmd.Name, exitCode)
	}
}

func executeFile(filename string, env *core.Environment) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	runInput(string(content), env, false)
}

func startRepl(env *core.Environment) {
	home, _ := os.UserHomeDir()
	historyFile := filepath.Join(home, ".kami_history")

	if *readlineLib == "base" || *readlineLib == "chzyer" {
		startChzyerRepl(env, historyFile)
	} else {
		// Default to wjq
		startWjqRepl(env, historyFile)
	}
}

func startChzyerRepl(env *core.Environment, historyFile string) {
	rl, err := chreadline.NewEx(&chreadline.Config{
		Prompt:          buildPrompt(false),
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
		rl.SetPrompt(buildPrompt(false))
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

func startWjqRepl(env *core.Environment, historyFile string) {
	cfg := &wjqreadline.Config{
		Prompt:    buildPrompt(true),
		Completer: &KamiCompleter{env: env},
		History:   NewWjqFileHistory(historyFile),
	}

	rl, err := wjqreadline.NewInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing wjq readline: %v\n", err)
		return
	}

	for {
		rl.SetPrompt(buildPrompt(true))
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

func buildPrompt(color bool) string {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		if color {
			return "\033[36mkami>\033[0m "
		}
		return "kami> "
	}

	name := filepath.Base(wd)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = wd
	}

	if color {
		return fmt.Sprintf("\033[90m[%s]\033[0m \033[36mkami>\033[0m ", name)
	}
	return fmt.Sprintf("[%s] kami> ", name)
}

func runInput(input string, env *core.Environment, isRepl bool) {
	l := core.NewLexer(input)
	p := core.NewParser(l)

	program := p.ParseProgram()
	result := core.Eval(program, env)
	if result != nil {
		if result.Type() == core.ERROR_OBJ {
			fmt.Fprintf(os.Stderr, "%s\n", result.Inspect())
		} else if isRepl && result.Type() != core.NULL_OBJ {
			fmt.Println(result.Inspect())
		}
	}
}

func loadConfig(env *core.Environment) {
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
