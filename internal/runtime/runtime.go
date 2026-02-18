package runtime

import (
	"fmt"
	"kamishell/internal/ast"
	"os"
	"os/exec"
)

func Eval(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, statement := range node.Statements {
			err := Eval(statement)
			if err != nil {
				return err
			}
		}
	case *ast.CommandStatement:
		cmd := exec.Command(node.Name, node.Arguments...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	case *ast.PrintStatement:
		val := evalExpression(node.Expression)
		fmt.Println(val)
	}
	return nil
}

func evalExpression(expr ast.Expression) string {
	switch expr := expr.(type) {
	case *ast.StringLiteral:
		return expr.Value
	}
	return ""
}
