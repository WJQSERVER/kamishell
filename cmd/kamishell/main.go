package main

import (
	"bufio"
	"fmt"
	"io"
	"kamishell/internal/lexer"
	"kamishell/internal/parser"
	"kamishell/internal/runtime"
	"os"
)

const PROMPT = "kami> "

func main() {
	env := runtime.NewEnvironment()

	if len(os.Args) > 1 {
		// Script mode
		filename := os.Args[1]
		executeFile(filename, env)
	} else {
		// REPL mode
		startRepl(os.Stdin, os.Stdout, env)
	}
}

func executeFile(filename string, env *runtime.Environment) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	runInput(string(content), env, false)
}

func startRepl(in io.Reader, out io.Writer, env *runtime.Environment) {
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
		if line == "exit" {
			break
		}

		runInput(line, env, true)
	}
}

func runInput(input string, env *runtime.Environment, isRepl bool) {
	l := lexer.New(input)
	p := parser.New(l)

	program := p.ParseProgram()
	result := runtime.Eval(program, env)
	if result != nil {
		if result.Type() == runtime.ERROR_OBJ {
			fmt.Fprintf(os.Stderr, "%s\n", result.Inspect())
		} else if isRepl && result.Type() != runtime.NULL_OBJ {
			fmt.Println(result.Inspect())
		}
	}
}
