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
	for {
		fmt.Print(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		if line == "exit" {
			break
		}

		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		err := runtime.Eval(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}
