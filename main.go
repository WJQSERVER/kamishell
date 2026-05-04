package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kamishell/builtin"
	"kamishell/core"
	"kamishell/make"
	"kamishell/recompiler"

	"github.com/WJQSERVER/readline"
)

func init() {
	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name:        "make",
		Description: "Build system inspired by CMake using .km scripts",
		Action:      make.Make,
	})
}

func main() {
	compileFlag := flag.String("compile", "", "Compile .km script to binary (specify output binary name)")
	sourceOnly := flag.String("source", "", "Compile .km script to Go source file only (specify output .go file)")
	flag.Parse()
	env := core.NewEnvironment()

	// Load .kamirc
	loadConfig(env)

	args := flag.Args()

	// Handle --compile mode
	if *compileFlag != "" || *sourceOnly != "" {
		scriptFile := ""
		outputFile := *compileFlag
		outputSource := *sourceOnly
		// Find the script file in remaining args
		for _, a := range args {
			if strings.HasSuffix(a, ".km") || strings.HasSuffix(a, ".kami") {
				scriptFile = a
			}
		}
		if scriptFile == "" {
			fmt.Fprintln(os.Stderr, "Error: no .km script file specified")
			os.Exit(1)
		}
		if outputFile == "" && outputSource == "" {
			outputFile = "app"
		}
		compileScript(scriptFile, outputFile, outputSource)
		return
	}

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
	if builtin.BuiltinHelpRequested(args[1:]) {
		builtin.PrintBuiltinHelp(cmd, os.Stdout)
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
	historyFile := resolveHistoryFile(os.UserHomeDir)
	runRepl(env, historyFile)
}

func runRepl(env *core.Environment, historyFile string) {
	cfg := &readline.Config{
		Prompt:    buildPrompt(true),
		Completer: &KamiCompleter{env: env},
		History:   NewFileHistory(historyFile),
	}

	rl, err := readline.NewInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		return
	}

	for {
		rl.SetPrompt(buildPrompt(true))
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
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
	loadConfigWithIO(env, os.Stderr, defaultConfigPaths)
}

func defaultConfigPaths() []string {
	return []string{
		os.ExpandEnv("$HOME/.kamirc"),
		".kamirc",
	}
}

func loadConfigWithIO(env *core.Environment, stderr io.Writer, paths func() []string) {
	for _, path := range paths() {
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(stderr, "Error reading config file %s: %v\n", path, err)
				continue
			}
			runInput(string(content), env, false)
		}
	}
}

func resolveHistoryFile(userHomeDir func() (string, error)) string {
	home, err := userHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".kami_history")
	}
	return ".kami_history"
}

type FileHistory struct {
	readline.History
	filepath string
}

func NewFileHistory(path string) *FileHistory {
	h := &FileHistory{
		History:  readline.NewHistory(),
		filepath: path,
	}
	if data, err := os.ReadFile(path); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if line != "" {
				h.History.Append(line)
			}
		}
	}
	return h
}

func (h *FileHistory) Append(line string) {
	h.History.Append(line)
	f, err := os.OpenFile(h.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		_ = f.Close()
		return
	}
	_ = f.Close()
}

func compileScript(scriptFile, binaryName, sourceOnly string) {
	content, err := os.ReadFile(scriptFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading script: %v\n", err)
		os.Exit(1)
	}

	lexer := core.NewLexer(string(content))
	parser := core.NewParser(lexer)
	program := parser.ParseProgram()

	// Check for parse errors
	for _, stmt := range program.Statements {
		if inv, ok := stmt.(*core.InvalidStatement); ok {
			fmt.Fprintf(os.Stderr, "Parse error: %s\n", inv.Message)
			os.Exit(1)
		}
	}

	compiled, err := recompiler.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// Format the generated Go code
	formatted, err := format.Source([]byte(compiled.Source))
	if err != nil {
		// Output unformatted with warning
		fmt.Fprintf(os.Stderr, "Warning: go format failed: %v\n", err)
		formatted = []byte(compiled.Source)
	}

	if sourceOnly != "" {
		if err := os.WriteFile(sourceOnly, formatted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing source: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated Go source: %s\n", sourceOnly)
		return
	}

	// Build binary
	tmpDir, err := os.MkdirTemp("", "kami-compile-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, formatted, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temp file: %v\n", err)
		os.Exit(1)
	}

	if binaryName == "" {
		binaryName = "app"
	}

	cmd := exec.Command("go", "build", "-o", binaryName, goFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled %s -> %s\n", scriptFile, binaryName)
}
