package core

import (
	"fmt"
	"io"
	"kamishell/builtin"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	NULL   = &Null{}
	TRUE   = &Boolean{Value: true}
	FALSE  = &Boolean{Value: false}
	ENVPKG = &Package{Name: "env"}
)

var NativeFns = make(map[string]*NativeFunction)

func init() {
	NativeFns["env.Get"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 1 {
				return &Error{Message: "env.Get() expects exactly one argument"}
			}
			arg, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "env.Get() argument must be a string"}
			}
			if val, ok := env.GetPackageValue("env", arg.Value); ok {
				return &String{Value: val}
			}
			return &String{Value: ""}
		},
	}

	NativeFns["env.Set"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 2 {
				return &Error{Message: "env.Set() expects exactly two arguments"}
			}

			name, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "env.Set() first argument must be a string"}
			}

			value, ok := objectToScriptString(args[1])
			if !ok {
				return &Error{Message: "env.Set() second argument must be string-compatible"}
			}

			env.SetPackageValue("env", name.Value, value)
			return &String{Value: value}
		},
	}

	NativeFns["env.Unset"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 1 {
				return &Error{Message: "env.Unset() expects exactly one argument"}
			}
			name, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "env.Unset() argument must be a string"}
			}
			return nativeBoolToBooleanObject(env.DeletePackageValue("env", name.Value))
		},
	}
}

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
	case *GoStatement:
		id := builtin.RegisterJob(node.String())
		go func() {
			EvalWithIO(node.Node, env, stdin, stdout, stderr)
			builtin.CompleteJob(id)
		}()
		return NULL
	case *BackgroundStatement:
		id := builtin.RegisterJob(node.String())
		go func() {
			EvalWithIO(node.Stmt, env, stdin, stdout, stderr)
			builtin.CompleteJob(id)
		}()
		return NULL
	case *FunctionStatement:
		return evalFunctionStatement(node, env)
	case *InfixExpression:
		left := EvalWithIO(node.Left, env, stdin, stdout, stderr)
		right := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		return evalInfixExpression(node.Operator, left, right)
	case *MemberExpression:
		return evalMemberExpression(node, env)
	case *CallExpression:
		return evalCallExpression(node, env, stdin, stdout, stderr)
	case *VarStatement:
		return evalVarStatement(node, env, stdin, stdout, stderr)
	case *AssignStatement:
		val := EvalWithIO(node.Value, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}

		expectedType, ok := env.GetType(node.Name.Value)
		if ok && expectedType != "" && string(val.Type()) != expectedType {
			return &Error{Message: fmt.Sprintf("cannot assign %s to variable of type %s", val.Type(), expectedType)}
		}

		if node.Token.Literal == ":=" {
			typeName := ""
			if shouldTrackType(string(val.Type())) {
				typeName = string(val.Type())
			}
			env.SetWithType(node.Name.Value, val, typeName)
		} else {
			env.Assign(node.Name.Value, val)
		}
		return val
	case *CommandStatement:
		return executeCommand(node.Name, node.Arguments, env, stdin, stdout, stderr)
	case *PrintStatement:
		val := EvalWithIO(node.Expression, env, stdin, stdout, stderr)
		fmt.Fprintln(stdout, inspectObject(val))
		return NULL
	case *ExecStatement:
		return evalExecStatement(node, env, stdin, stdout, stderr)
	case *Identifier:
		return evalIdentifier(node, env)
	case *StringLiteral:
		if node.Obj != nil && strings.IndexByte(node.Value, '$') < 0 {
			return node.Obj
		}
		return &String{Value: os.Expand(node.Value, func(name string) string {
			if obj, ok := env.GetObject(name); ok {
				return inspectObject(obj)
			}
			return os.Getenv(name)
		})}
	case *IntegerLiteral:
		if node.Obj != nil {
			return node.Obj
		}
		return getIntegerObject(node.Value)
	case *NilLiteral:
		return NULL
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
			env.Set("err", errObj)
		} else {
			env.Set("err", NULL)
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
	if operator == "+" {
		if left.Type() == STRING_OBJ || right.Type() == STRING_OBJ {
			return &String{Value: inspectObject(left) + inspectObject(right)}
		}
	}

	switch {
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(inspectObject(left) == inspectObject(right))
	case operator == "!=":
		return nativeBoolToBooleanObject(inspectObject(left) != inspectObject(right))
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
		return getIntegerObject(leftVal + rightVal)
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
		return &Error{Message: err.Error(), Code: 1, Op: "redirect"}
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
	// Eval arguments first
	evaledArgs := make([]Object, 0, len(args))
	for _, arg := range args {
		val := EvalWithIO(arg, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		evaledArgs = append(evaledArgs, val)
	}

	// 1. Check for native functions (global)
	if fn, ok := NativeFns[name]; ok {
		return fn.Fn(env, evaledArgs...)
	}

	// 2. Check for user-defined functions or native functions in env
	if val, ok := env.GetObject(name); ok {
		if fn, ok := val.(*Function); ok {
			// Convert []Object to []string for user funcs
			strArgs := inspectObjects(evaledArgs)
			return applyFunction(fn, strArgs, env, stdin, stdout, stderr)
		}
		if fn, ok := val.(*NativeFunction); ok {
			return fn.Fn(env, evaledArgs...)
		}
	}

	// Convert []Object to []string for builtins and external commands
	strArgs := inspectObjects(evaledArgs)

	// 3. Check for builtins
	if cmd, ok := builtin.Builtins[name]; ok {
		exitCode := cmd.Action(strArgs, env, stdin, stdout, stderr)
		if exitCode != 0 {
			return &Error{Message: fmt.Sprintf("builtin %s failed", name), Code: exitCode, Op: name}
		}
		return NULL
	}

	// 4. External command
	cmd := exec.Command(name, strArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return &Error{Message: err.Error(), Code: exitError.ExitCode(), Op: name}
		}
		return &Error{Message: err.Error(), Code: 1, Op: name}
	}
	return NULL
}

func evalMemberExpression(node *MemberExpression, env *Environment) Object {
	if ident, ok := node.Object.(*Identifier); ok && ident.Value == "env" {
		name := "env." + node.Property.Value
		if fn, ok := NativeFns[name]; ok {
			return fn
		}
		if val, ok := env.GetObject(name); ok {
			return val
		}
		return &Error{Message: "member not found: " + name}
	}

	left := EvalWithIO(node.Object, env, os.Stdin, os.Stdout, os.Stderr)
	if isError(left) {
		return left
	}

	name := inspectObject(left) + "." + node.Property.Value
	if fn, ok := NativeFns[name]; ok {
		return fn
	}
	if val, ok := env.GetObject(name); ok {
		return val
	}

	return &Error{Message: "member not found: " + name}
}

func evalCallExpression(node *CallExpression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	function := EvalWithIO(node.Function, env, stdin, stdout, stderr)
	if isError(function) {
		return function
	}

	args := make([]Object, 0, len(node.Arguments))
	for _, arg := range node.Arguments {
		value := EvalWithIO(arg, env, stdin, stdout, stderr)
		if isError(value) {
			return value
		}
		args = append(args, value)
	}

	switch fn := function.(type) {
	case *NativeFunction:
		return fn.Fn(env, args...)
	case *Function:
		strArgs := inspectObjects(args)
		return applyFunction(fn, strArgs, env, stdin, stdout, stderr)
	default:
		return &Error{Message: fmt.Sprintf("not callable: %s", function.Type())}
	}
}

func executeCommandWithStrings(name string, args []string, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	// This function is now mostly for builtins and external commands.
	// We need to handle native/user functions with evaluated objects.
	// For simplicity, let's keep the string-based logic for now.

	if val, ok := env.GetObject(name); ok {
		if fn, ok := val.(*Function); ok {
			return applyFunction(fn, args, env, stdin, stdout, stderr)
		}
	}

	if cmd, ok := builtin.Builtins[name]; ok {
		exitCode := cmd.Action(args, env, stdin, stdout, stderr)
		if exitCode != 0 {
			return &Error{Message: fmt.Sprintf("builtin %s failed", name), Code: exitCode, Op: name}
		}
		return NULL
	}

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return &Error{Message: err.Error(), Code: exitError.ExitCode(), Op: name}
		}
		return &Error{Message: err.Error(), Code: 1, Op: name}
	}
	return NULL
}

func evalIdentifier(node *Identifier, env *Environment) Object {
	if node.Value == "env" {
		return ENVPKG
	}
	if fn, ok := NativeFns[node.Value]; ok {
		return fn
	}
	val, ok := env.GetObject(node.Value)
	if !ok {
		return &Error{Message: "identifier not found: " + node.Value}
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

func objectToScriptString(obj Object) (string, bool) {
	switch obj.Type() {
	case STRING_OBJ, INTEGER_OBJ, BOOLEAN_OBJ, NULL_OBJ:
		return inspectObject(obj), true
	default:
		return "", false
	}
}

func inspectObjects(args []Object) []string {
	if len(args) == 0 {
		return nil
	}
	values := make([]string, len(args))
	for i, arg := range args {
		values[i] = inspectObject(arg)
	}
	return values
}

func inspectObject(obj Object) string {
	switch v := obj.(type) {
	case *String:
		return v.Value
	case *Integer:
		return integerToString(v.Value)
	case *Boolean:
		if v.Value {
			return "true"
		}
		return "false"
	case *Null:
		return "nil"
	default:
		return obj.Inspect()
	}
}

func integerToString(value int64) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	idx := len(buf)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		idx--
		buf[idx] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		idx--
		buf[idx] = '-'
	}
	return string(buf[idx:])
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

func evalVarStatement(vs *VarStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	typeName := ""
	if vs.Type != nil {
		typeName = string(mapTypeName(vs.Type.Value))
	}

	val, errObj := evaluateDeclaredValue(vs, env, stdin, stdout, stderr, typeName)
	if errObj != nil {
		return errObj
	}

	if typeName == "" && shouldTrackType(string(val.Type())) {
		typeName = string(val.Type())
	}

	env.SetWithType(vs.Name.Value, val, typeName)
	return val
}

func evaluateDeclaredValue(vs *VarStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer, typeName string) (Object, *Error) {
	if vs.Value == nil {
		if typeName == "" {
			return NULL, nil
		}
		return zeroValueForType(ObjectType(typeName)), nil
	}

	val := EvalWithIO(vs.Value, env, stdin, stdout, stderr)
	if isError(val) {
		return nil, val.(*Error)
	}

	if typeName != "" && string(val.Type()) != typeName {
		return nil, &Error{Message: fmt.Sprintf("cannot initialize %s with value of type %s", typeName, val.Type())}
	}

	return val, nil
}

func mapTypeName(name string) ObjectType {
	switch strings.ToLower(name) {
	case "int", "integer":
		return INTEGER_OBJ
	case "string":
		return STRING_OBJ
	case "bool", "boolean":
		return BOOLEAN_OBJ
	default:
		return ObjectType(strings.ToUpper(name))
	}
}

func zeroValueForType(typeName ObjectType) Object {
	switch typeName {
	case INTEGER_OBJ:
		return getIntegerObject(0)
	case STRING_OBJ:
		return &String{Value: ""}
	case BOOLEAN_OBJ:
		return FALSE
	default:
		return NULL
	}
}
