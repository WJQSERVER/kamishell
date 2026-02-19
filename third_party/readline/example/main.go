package main

import (
	"fmt"
	"io"
	"github.com/WJQSERVER/readline"
)

func main() {
	completer := &readline.PrefixCompleter{
		Candidates: []string{"help", "exit", "list", "show", "hello", "你好"},
	}

	rl, err := readline.NewInstance(&readline.Config{
		Prompt:    "> ",
		Completer: completer,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	fmt.Println("Pure Go Readline Example (Type 'exit' or Ctrl-D to quit)")

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			} else if err == io.EOF {
				break
			}
			fmt.Println("Error:", err)
			break
		}

		fmt.Printf("You typed: %s\n", line)
		if line == "exit" {
			break
		}
	}
}
