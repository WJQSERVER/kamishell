package main

import (
	"bufio"
	"fmt"
	"io"
	"kamishell"
	"os"
)

const PROMPT = "kami> "

func main() {
	env := kamishell.NewEnvironment()

	if len(os.Args) > 1 {
		// Script mode
		filename := os.Args[1]
		executeFile(filename, env)
	} else {
		// REPL mode
		startRepl(os.Stdin, os.Stdout, env)
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

func startRepl(in io.Reader, out io.Writer, env *kamishell.Environment) {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		if line == "" {
			continue
		}
		// exit handled by builtin

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
