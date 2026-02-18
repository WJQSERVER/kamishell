package runtime

import (
	"fmt"
	"kamishell/internal/ast"
	"os"
	"os/exec"
	"strings"
)

var (
	NULL  = &Null{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalStatements(node.Statements, env)
	case *ast.BlockStatement:
		return evalStatements(node.Statements, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.IfStatement:
		return evalIfStatement(node, env)
	case *ast.AssignStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val
	case *ast.CommandStatement:
		return executeCommand(node.Name, node.Arguments)
	case *ast.PrintStatement:
		val := Eval(node.Expression, env)
		if isError(val) {
			return val
		}
		fmt.Println(val.Inspect())
		return NULL
	case *ast.ExecStatement:
		return evalExecStatement(node, env)
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.StringLiteral:
		return &String{Value: node.Value}
	case *ast.IntegerLiteral:
		return &Integer{Value: node.Value}
	case *ast.BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)
	}
	return NULL
}

func evalStatements(stmts []ast.Statement, env *Environment) Object {
	var result Object
	for _, statement := range stmts {
		result = Eval(statement, env)
		if errObj, ok := result.(*Error); ok {
			return errObj
		}
	}
	return result
}

func evalIfStatement(is *ast.IfStatement, env *Environment) Object {
	condition := Eval(is.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(is.Consequence, env)
	} else if is.Alternative != nil {
		return Eval(is.Alternative, env)
	} else {
		return NULL
	}
}

func evalExecStatement(es *ast.ExecStatement, env *Environment) Object {
	val := Eval(es.CommandStr, env)
	if isError(val) {
		return val
	}
	cmdStr, ok := val.(*String)
	if !ok {
		return &Error{Message: "exec expects a string"}
	}

	fields := strings.Fields(cmdStr.Value)
	if len(fields) == 0 {
		return NULL
	}

	return executeCommand(fields[0], fields[1:])
}

func isTruthy(obj Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		if i, ok := obj.(*Integer); ok && i.Value == 0 {
			return false
		}
		return true
	}
}

func executeCommand(name string, args []string) Object {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return &Error{Message: err.Error()}
	}
	return NULL
}

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return &Error{Message: fmt.Sprintf("identifier not found: %s", node.Value)}
	}
	return val
}

func nativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}
