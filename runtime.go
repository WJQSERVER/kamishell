package kamishell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"kamishell/builtin"
)

var (
	NULL  = &Null{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

func Eval(node Node, env *Environment) Object {
	return EvalWithIO(node, env, os.Stdin, os.Stdout, os.Stderr)
}

func EvalWithIO(node Node, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	switch node := node.(type) {
	case *Program:
		return evalStatements(node.Statements, env, stdin, stdout, stderr)
	case *BlockStatement:
		return evalStatements(node.Statements, env, stdin, stdout, stderr)
	case *ExpressionStatement:
		return EvalWithIO(node.Expression, env, stdin, stdout, stderr)
	case *IfStatement:
		return evalIfStatement(node, env, stdin, stdout, stderr)
	case *ForStatement:
		return evalForStatement(node, env, stdin, stdout, stderr)
	case *PipeStatement:
		return evalPipeStatement(node, env, stdin, stdout, stderr)
	case *RedirectStatement:
		return evalRedirectStatement(node, env, stdin, stdout, stderr)
	case *LogicalStatement:
		return evalLogicalStatement(node, env, stdin, stdout, stderr)
	case *FunctionStatement:
		return evalFunctionStatement(node, env)
	case *InfixExpression:
		left := EvalWithIO(node.Left, env, stdin, stdout, stderr)
		if isError(left) {
			return left
		}
		right := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *AssignStatement:
		val := EvalWithIO(node.Value, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		return val
	case *CommandStatement:
		return executeCommand(node.Name, node.Arguments, env, stdin, stdout, stderr)
	case *PrintStatement:
		val := EvalWithIO(node.Expression, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		fmt.Fprintln(stdout, val.Inspect())
		return NULL
	case *ExecStatement:
		return evalExecStatement(node, env, stdin, stdout, stderr)
	case *Identifier:
		return evalIdentifier(node, env)
	case *StringLiteral:
		return &String{Value: os.Expand(node.Value, func(name string) string {
			if val, ok := env.Get(name); ok {
				if obj, ok := val.(Object); ok {
					return obj.Inspect()
				}
			}
			return os.Getenv(name)
		})}
	case *IntegerLiteral:
		return &Integer{Value: node.Value}
	case *BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)
	}
	return NULL
}

func evalStatements(stmts []Statement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var result Object
	for _, statement := range stmts {
		result = EvalWithIO(statement, env, stdin, stdout, stderr)
		if errObj, ok := result.(*Error); ok {
			return errObj
		}
	}
	return result
}

func evalIfStatement(is *IfStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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

func evalForStatement(fs *ForStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
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

func evalStringInfixExpression(operator string, left, right Object) Object {
	if operator != "+" {
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", left.Type(), operator, right.Type())}
	}
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value
	return &String{Value: leftVal + rightVal}
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

func evalExecStatement(es *ExecStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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

func evalPipeStatement(ps *PipeStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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

func evalRedirectStatement(rs *RedirectStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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

func executeCommand(name string, args []Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
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
	if val, ok := env.Get(name); ok {
		if fn, ok := val.(*Function); ok {
			return applyFunction(fn, args, env, stdin, stdout, stderr)
		}
	}

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

func evalIdentifier(node *Identifier, env *Environment) Object {
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

func evalLogicalStatement(ls *LogicalStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	left := EvalWithIO(ls.Left, env, stdin, stdout, stderr)

	if ls.Operator == "&&" {
		if !isError(left) {
			return EvalWithIO(ls.Right, env, stdin, stdout, stderr)
		}
		return left
	} else if ls.Operator == "||" {
		if isError(left) {
			return EvalWithIO(ls.Right, env, stdin, stdout, stderr)
		}
		return left
	}

	return &Error{Message: fmt.Sprintf("unknown logical operator: %s", ls.Operator)}
}

func evalFunctionStatement(fs *FunctionStatement, env *Environment) Object {
	fn := &Function{
		Parameters: fs.Parameters,
		Body:       fs.Body,
		Env:        env,
	}
	env.Set(fs.Name.Value, fn)
	return NULL
}

func applyFunction(fn *Function, args []string, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	extendEnv := NewEnclosedEnvironment(fn.Env)

	for i, param := range fn.Parameters {
		if i < len(args) {
			extendEnv.Set(param.Value, &String{Value: args[i]})
		}
	}

	return EvalWithIO(fn.Body, extendEnv, stdin, stdout, stderr)
}
