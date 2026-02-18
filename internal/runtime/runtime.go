package runtime

import (
	"fmt"
	"io"
	"kamishell/internal/ast"
	"kamishell/internal/builtin"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	NULL  = &Null{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

func Eval(node ast.Node, env *Environment) Object {
	return EvalWithIO(node, env, os.Stdin, os.Stdout, os.Stderr)
}

func EvalWithIO(node ast.Node, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalStatements(node.Statements, env, stdin, stdout, stderr)
	case *ast.BlockStatement:
		return evalStatements(node.Statements, env, stdin, stdout, stderr)
	case *ast.ExpressionStatement:
		return EvalWithIO(node.Expression, env, stdin, stdout, stderr)
	case *ast.IfStatement:
		return evalIfStatement(node, env, stdin, stdout, stderr)
	case *ast.ForStatement:
		return evalForStatement(node, env, stdin, stdout, stderr)
	case *ast.PipeStatement:
		return evalPipeStatement(node, env, stdin, stdout, stderr)
	case *ast.RedirectStatement:
		return evalRedirectStatement(node, env, stdin, stdout, stderr)
	case *ast.InfixExpression:
		left := EvalWithIO(node.Left, env, stdin, stdout, stderr)
		if isError(left) {
			return left
		}
		right := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *ast.AssignStatement:
		val := EvalWithIO(node.Value, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val
	case *ast.CommandStatement:
		return executeCommand(node.Name, node.Arguments, env, stdin, stdout, stderr)
	case *ast.PrintStatement:
		val := EvalWithIO(node.Expression, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		fmt.Fprintln(stdout, val.Inspect())
		return NULL
	case *ast.ExecStatement:
		return evalExecStatement(node, env, stdin, stdout, stderr)
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

func evalStatements(stmts []ast.Statement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var result Object
	for _, statement := range stmts {
		result = EvalWithIO(statement, env, stdin, stdout, stderr)
		if errObj, ok := result.(*Error); ok {
			return errObj
		}
	}
	return result
}

func evalIfStatement(is *ast.IfStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	condition := EvalWithIO(is.Condition, env, stdin, stdout, stderr)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return EvalWithIO(is.Consequence, env, stdin, stdout, stderr)
	} else if is.Alternative != nil {
		return EvalWithIO(is.Alternative, env, stdin, stdout, stderr)
	} else {
		return NULL
	}
}

func evalForStatement(fs *ast.ForStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var result Object = NULL
	for {
		if fs.Condition != nil {
			condition := EvalWithIO(fs.Condition, env, stdin, stdout, stderr)
			if isError(condition) {
				return condition
			}
			if !isTruthy(condition) {
				break
			}
		}

		result = EvalWithIO(fs.Consequence, env, stdin, stdout, stderr)
		if isError(result) {
			return result
		}

		if fs.Condition == nil {
			break
		}
	}
	return result
}

func evalInfixExpression(operator string, left, right Object) Object {
	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left.Inspect() == right.Inspect())
	case operator == "!=":
		return nativeBoolToBooleanObject(left.Inspect() != right.Inspect())
	case left.Type() != right.Type():
		return &Error{Message: fmt.Sprintf("type mismatch: %s %s %s", left.Type(), operator, right.Type())}
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", left.Type(), operator, right.Type())}
	}
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case "+":
		return &Integer{Value: leftVal + rightVal}
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", left.Type(), operator, right.Type())}
	}
}

func evalExecStatement(es *ast.ExecStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	val := EvalWithIO(es.CommandStr, env, stdin, stdout, stderr)
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

	return executeCommandWithStrings(fields[0], fields[1:], env, stdin, stdout, stderr)
}

func evalPipeStatement(ps *ast.PipeStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	n := len(ps.Commands)
	pipes := make([]*io.PipeWriter, n-1)
	readers := make([]*io.PipeReader, n-1)

	for i := 0; i < n-1; i++ {
		readers[i], pipes[i] = io.Pipe()
	}

	var wg sync.WaitGroup
	wg.Add(n)

	var errs []string
	var errMu sync.Mutex

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()

			var curStdin io.Reader
			var curStdout io.Writer

			if idx == 0 {
				curStdin = stdin
			} else {
				curStdin = readers[idx-1]
			}

			if idx == n-1 {
				curStdout = stdout
			} else {
				curStdout = pipes[idx]
			}

			res := EvalWithIO(ps.Commands[idx], env, curStdin, curStdout, stderr)

			if idx < n-1 {
				pipes[idx].Close()
			}
			if idx > 0 {
				readers[idx-1].Close()
			}

			if isError(res) {
				errMu.Lock()
				errs = append(errs, res.Inspect())
				errMu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		return &Error{Message: strings.Join(errs, "; ")}
	}

	return NULL
}

func evalRedirectStatement(rs *ast.RedirectStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	target := EvalWithIO(rs.Target, env, stdin, stdout, stderr)
	if isError(target) {
		return target
	}

	path, ok := target.(*String)
	if !ok {
		return &Error{Message: "redirection target must be a string"}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if rs.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(path.Value, flags, 0644)
	if err != nil {
		return &Error{Message: err.Error()}
	}
	defer f.Close()

	return EvalWithIO(rs.Source, env, stdin, f, stderr)
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

func executeCommand(name string, args []ast.Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	evaledArgs := make([]string, len(args))
	for i, arg := range args {
		val := EvalWithIO(arg, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		evaledArgs[i] = val.Inspect()
	}
	return executeCommandWithStrings(name, evaledArgs, env, stdin, stdout, stderr)
}

func executeCommandWithStrings(name string, args []string, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if fn, ok := builtin.Builtins[name]; ok {
		exitCode := fn(args, env, stdin, stdout, stderr)
		if exitCode != 0 {
			return &Error{Message: fmt.Sprintf("builtin %s exited with %d", name, exitCode)}
		}
		return NULL
	}

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
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
	return val.(Object)
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
