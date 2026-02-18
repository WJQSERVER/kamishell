package main

import (
	"bufio"
	"fmt"
	"kamishell/internal/lexer"
	"kamishell/internal/parser"
	"kamishell/internal/runtime"
	"os"
)

const PROMPT = "kami> "

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	env := runtime.NewEnvironment()

	for {
		fmt.Print(PROMPT)
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

		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		result := runtime.Eval(program, env)
		if result != nil {
			if result.Type() == runtime.ERROR_OBJ {
				fmt.Fprintf(os.Stderr, "%s\n", result.Inspect())
			} else if result.Type() != runtime.NULL_OBJ {
				fmt.Println(result.Inspect())
			}
		}
	}
}
