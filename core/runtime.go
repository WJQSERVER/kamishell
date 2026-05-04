package core

import (
	"bytes"
	"fmt"
	"io"
	"kamishell/builtin"
	"math"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	NULL    = &Null{}
	TRUE    = &Boolean{Value: true}
	FALSE   = &Boolean{Value: false}
	ENVPKG  = &Package{Name: "env"}
	SYNCPKG = &Package{Name: "sync"}
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

	NativeFns["env.GetOS"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 0 {
				return &Error{Message: "env.GetOS() expects no arguments"}
			}
			return &String{Value: builtin.GetOS()}
		},
	}

	NativeFns["env.GetArch"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 0 {
				return &Error{Message: "env.GetArch() expects no arguments"}
			}
			return &String{Value: builtin.GetArch()}
		},
	}

	NativeFns["param.Get"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 1 {
				return &Error{Message: "param.Get() expects exactly one argument"}
			}
			key, ok := args[0].(*String)
			if !ok {
				return &Error{Message: "param.Get() argument must be a string"}
			}
			if val, ok := env.GetObject("param." + key.Value); ok {
				return val
			}
			return NULL
		},
	}

	NativeFns["len"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 1 {
				return &Error{Message: "len() expects exactly one argument"}
			}
			switch v := args[0].(type) {
			case *Array:
				return getIntegerObject(int64(len(v.Elements)))
			case *String:
				return getIntegerObject(int64(len(v.Value)))
			default:
				return &Error{Message: fmt.Sprintf("len() not supported for type %s", v.Type())}
			}
		},
	}

	NativeFns["push"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) != 2 {
				return &Error{Message: "push() expects exactly two arguments (array, element)"}
			}
			arr, ok := args[0].(*Array)
			if !ok {
				return &Error{Message: "push() first argument must be an array"}
			}
			elem := args[1]
			if elem.Type() != arr.ElemType {
				return &Error{Message: fmt.Sprintf("push() type mismatch: cannot push %s into ARRAY[%s]", elem.Type(), arr.ElemType)}
			}
			newElems := make([]Object, len(arr.Elements)+1)
			copy(newElems, arr.Elements)
			newElems[len(arr.Elements)] = elem
			return &Array{ElemType: arr.ElemType, Elements: newElems}
		},
	}

	// error() — create an Error object
	NativeFns["error"] = &NativeFunction{
		Fn: func(env *Environment, args ...Object) Object {
			if len(args) == 0 {
				return &Error{Message: ""}
			}
			msg := ""
			if s, ok := args[0].(*String); ok {
				msg = s.Value
			} else {
				msg = args[0].Inspect()
			}
			return &Error{Message: msg}
		},
	}
}

// Go标准库映射表
var goStdlib = map[string]map[string]*NativeFunction{
	"fmt": {
		"Println": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				// 转换参数并调用fmt.Println
				goArgs := make([]any, len(args))
				for i, arg := range args {
					switch v := arg.(type) {
					case *Integer:
						goArgs[i] = v.Value
					case *String:
						goArgs[i] = v.Value
					case *Boolean:
						goArgs[i] = v.Value
					default:
						goArgs[i] = arg.Inspect()
					}
				}
				fmt.Println(goArgs...)
				return NULL
			},
		},
		"Printf": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) < 1 {
					return &Error{Message: "Printf requires at least one argument"}
				}
				format, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Printf first argument must be a string"}
				}
				// 转换剩余参数
				goArgs := make([]any, len(args)-1)
				for i := 1; i < len(args); i++ {
					switch v := args[i].(type) {
					case *Integer:
						goArgs[i-1] = v.Value
					case *String:
						goArgs[i-1] = v.Value
					case *Boolean:
						goArgs[i-1] = v.Value
					default:
						goArgs[i-1] = args[i].Inspect()
					}
				}
				fmt.Printf(format.Value, goArgs...)
				return NULL
			},
		},
		"Sprintf": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) < 1 {
					return &Error{Message: "Sprintf requires at least one argument"}
				}
				format, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Sprintf first argument must be a string"}
				}
				// 转换剩余参数
				goArgs := make([]any, len(args)-1)
				for i := 1; i < len(args); i++ {
					switch v := args[i].(type) {
					case *Integer:
						goArgs[i-1] = v.Value
					case *String:
						goArgs[i-1] = v.Value
					case *Boolean:
						goArgs[i-1] = v.Value
					default:
						goArgs[i-1] = args[i].Inspect()
					}
				}
				result := fmt.Sprintf(format.Value, goArgs...)
				return &String{Value: result}
			},
		},
	},
	"math": {
		"Sqrt": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return &Error{Message: "Sqrt requires exactly one argument"}
				}
				var x float64
				switch v := args[0].(type) {
				case *Integer:
					x = float64(v.Value)
				case *Float:
					x = v.Value
				default:
					return &Error{Message: "Sqrt argument must be a number"}
				}
				if x < 0 {
					return &Error{Message: "Sqrt argument must be non-negative"}
				}
				return &Float{Value: math.Sqrt(x)}
			},
		},
		"Abs": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return &Error{Message: "Abs requires exactly one argument"}
				}
				switch v := args[0].(type) {
				case *Integer:
					if v.Value < 0 {
						return &Integer{Value: -v.Value}
					}
					return v
				case *Float:
					if v.Value < 0 {
						return &Float{Value: -v.Value}
					}
					return v
				default:
					return &Error{Message: "Abs argument must be a number"}
				}
			},
		},
	},
	"strings": {
		"Contains": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "Contains requires exactly two arguments"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Contains first argument must be a string"}
				}
				substr, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "Contains second argument must be a string"}
				}
				return nativeBoolToBooleanObject(strings.Contains(s.Value, substr.Value))
			},
		},
		"HasPrefix": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "HasPrefix requires exactly two arguments"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "HasPrefix first argument must be a string"}
				}
				prefix, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "HasPrefix second argument must be a string"}
				}
				return nativeBoolToBooleanObject(strings.HasPrefix(s.Value, prefix.Value))
			},
		},
		"HasSuffix": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "HasSuffix requires exactly two arguments"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "HasSuffix first argument must be a string"}
				}
				suffix, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "HasSuffix second argument must be a string"}
				}
				return nativeBoolToBooleanObject(strings.HasSuffix(s.Value, suffix.Value))
			},
		},
		"Replace": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 3 {
					return &Error{Message: "Replace requires exactly three arguments"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Replace first argument must be a string"}
				}
				old, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "Replace second argument must be a string"}
				}
				new, ok := args[2].(*String)
				if !ok {
					return &Error{Message: "Replace third argument must be a string"}
				}
				return &String{Value: strings.Replace(s.Value, old.Value, new.Value, -1)}
			},
		},
		"Split": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "Split requires exactly two arguments"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Split first argument must be a string"}
				}
				sep, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "Split second argument must be a string"}
				}
				parts := strings.Split(s.Value, sep.Value)
				// 返回字符串数组（暂时用字符串表示）
				var result strings.Builder
				result.WriteString("[")
				for i, part := range parts {
					if i > 0 {
						result.WriteString(", ")
					}
					result.WriteString("\"" + part + "\"")
				}
				result.WriteString("]")
				return &String{Value: result.String()}
			},
		},
		"Join": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "Join requires exactly two arguments"}
				}
				arr, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Join first argument must be a string"}
				}
				sep, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "Join second argument must be a string"}
				}
				return &String{Value: strings.Join([]string{arr.Value}, sep.Value)}
			},
		},
	},
	"strconv": {
		"Itoa": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return &Error{Message: "Itoa requires exactly one argument"}
				}
				switch v := args[0].(type) {
				case *Integer:
					return &String{Value: fmt.Sprintf("%d", v.Value)}
				default:
					return &Error{Message: "Itoa argument must be an integer"}
				}
			},
		},
		"Atoi": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return &Error{Message: "Atoi requires exactly one argument"}
				}
				s, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Atoi argument must be a string"}
				}
				var n int64
				_, err := fmt.Sscanf(s.Value, "%d", &n)
				if err != nil {
					return &Error{Message: "Atoi: invalid integer string"}
				}
				return &Integer{Value: n}
			},
		},
	},
	"os": {
		"Getenv": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return &Error{Message: "Getenv requires exactly one argument"}
				}
				key, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Getenv argument must be a string"}
				}
				return &String{Value: os.Getenv(key.Value)}
			},
		},
		"Setenv": &NativeFunction{
			Fn: func(env *Environment, args ...Object) Object {
				if len(args) != 2 {
					return &Error{Message: "Setenv requires exactly two arguments"}
				}
				key, ok := args[0].(*String)
				if !ok {
					return &Error{Message: "Setenv first argument must be a string"}
				}
				value, ok := args[1].(*String)
				if !ok {
					return &Error{Message: "Setenv second argument must be a string"}
				}
				err := os.Setenv(key.Value, value.Value)
				if err != nil {
					return &Error{Message: err.Error()}
				}
				return NULL
			},
		},
	},
}

func Eval(node Node, env *Environment) Object {
	return EvalWithIO(node, env, os.Stdin, os.Stdout, os.Stderr)
}

func EvalWithIO(node Node, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	switch node := node.(type) {
	case *Program:
		result := evalStatements(node.Statements, env, stdin, stdout, stderr)
		if returnValue, ok := result.(*ReturnValue); ok {
			return returnValue.Value
		}
		return result
	case *BlockStatement:
		return evalStatements(node.Statements, env, stdin, stdout, stderr)
	case *ExpressionStatement:
		return EvalWithIO(node.Expression, env, stdin, stdout, stderr)
	case *InvalidStatement:
		return &Error{Message: node.Message}
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
		asyncEnv := env.Clone()
		task := &Task{ID: id, Done: make(chan struct{})}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					task.Result = &Error{Message: fmt.Sprintf("goroutine panic: %v", r)}
					close(task.Done)
					builtin.CompleteJobWithResult(id, false, fmt.Sprintf("panic: %v", r))
				}
			}()
			result := EvalWithIO(node.Node, asyncEnv, stdin, stdout, stderr)
			// Unwrap ReturnValue to get the actual value
			if rv, ok := result.(*ReturnValue); ok {
				task.Result = rv.Value
			} else {
				task.Result = result
			}
			close(task.Done)
			if isError(result) {
				builtin.CompleteJobWithResult(id, false, result.Inspect())
				return
			}
			builtin.CompleteJobWithResult(id, true, "")
		}()
		return task
	case *GoExpression:
		id := builtin.RegisterJob(node.String())
		asyncEnv := env.Clone()
		task := &Task{ID: id, Done: make(chan struct{})}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					task.Result = &Error{Message: fmt.Sprintf("goroutine panic: %v", r)}
					close(task.Done)
					builtin.CompleteJobWithResult(id, false, fmt.Sprintf("panic: %v", r))
				}
			}()
			result := EvalWithIO(node.Node, asyncEnv, stdin, stdout, stderr)
			// Unwrap ReturnValue to get the actual value
			if rv, ok := result.(*ReturnValue); ok {
				task.Result = rv.Value
			} else {
				task.Result = result
			}
			close(task.Done)
			if isError(result) {
				builtin.CompleteJobWithResult(id, false, result.Inspect())
				return
			}
			builtin.CompleteJobWithResult(id, true, "")
		}()
		return task
	case *BackgroundStatement:
		id := builtin.RegisterJob(node.String())
		asyncEnv := env.Clone()
		go func() {
			result := EvalWithIO(node.Stmt, asyncEnv, stdin, stdout, stderr)
			if isError(result) {
				builtin.CompleteJobWithResult(id, false, result.Inspect())
				return
			}
			builtin.CompleteJobWithResult(id, true, "")
		}()
		return NULL
	case *FunctionStatement:
		return evalFunctionStatement(node, env)
	case *ImportStatement:
		return evalImportStatement(node, env)
	case *MethodCallBlockStatement:
		return evalMethodCallBlock(node, env, stdin, stdout, stderr)
	case *WaitStatement:
		return evalWaitStatement(node, env, stdin, stdout, stderr)
	case *SwitchStatement:
		return evalSwitchStatement(node, env, stdin, stdout, stderr)
	case *ReturnStatement:
		return evalReturnStatement(node, env, stdin, stdout, stderr)
	case *BreakStatement:
		return BREAK_SIGNAL
	case *ContinueStatement:
		return CONTINUE_SIGNAL
	case *InfixExpression:
		left := EvalWithIO(node.Left, env, stdin, stdout, stderr)
		right := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		return evalInfixExpression(node.Operator, left, right)
	case *PrefixExpression:
		return evalPrefixExpression(node, env, stdin, stdout, stderr)
	case *MemberExpression:
		return evalMemberExpression(node, env)
	case *CallExpression:
		return evalCallExpression(node, env, stdin, stdout, stderr)
	case *ArrayLiteral:
		return evalArrayLiteral(node, env, stdin, stdout, stderr)
	case *IndexExpression:
		return evalIndexExpression(node, env, stdin, stdout, stderr)
	case *VarStatement:
		return evalVarStatement(node, env, stdin, stdout, stderr)
	case *PointerAssignStatement:
		return evalPointerAssignStatement(node, env, stdin, stdout, stderr)
	case *AssignStatement:
		val := EvalWithIO(node.Value, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}

		// Index assignment: arr[i] = val
		if node.Target != nil {
			idxExpr, ok := node.Target.(*IndexExpression)
			if !ok {
				return &Error{Message: "invalid assignment target"}
			}
			left := EvalWithIO(idxExpr.Left, env, stdin, stdout, stderr)
			if isError(left) {
				return left
			}
			arr, ok := left.(*Array)
			if !ok {
				return &Error{Message: fmt.Sprintf("cannot index non-array type %s", left.Type())}
			}
			index := EvalWithIO(idxExpr.Index, env, stdin, stdout, stderr)
			if isError(index) {
				return index
			}
			idx, ok := index.(*Integer)
			if !ok {
				return &Error{Message: fmt.Sprintf("array index must be INTEGER, got %s", index.Type())}
			}
			i := idx.Value
			if i < 0 || i >= int64(len(arr.Elements)) {
				return &Error{Message: fmt.Sprintf("array index out of bounds: index %d, length %d", i, len(arr.Elements))}
			}
			if val.Type() != arr.ElemType {
				return &Error{Message: fmt.Sprintf("cannot assign %s to ARRAY[%s] element", val.Type(), arr.ElemType)}
			}
			arr.Elements[i] = val
			return val
		}

		// Multi-value assignment: val, err := div(10, 0)
		if len(node.Names) > 1 {
			tuple, ok := val.(*Tuple)
			if !ok {
				return &Error{Message: "cannot unpack non-tuple value into multiple variables"}
			}
			if len(tuple.Elements) != len(node.Names) {
				return &Error{Message: fmt.Sprintf("expected %d return values, got %d", len(node.Names), len(tuple.Elements))}
			}
			for i, name := range node.Names {
				elem := tuple.Elements[i]
				if node.Token.Literal == ":=" {
					typeName := ""
					if shouldTrackType(string(elem.Type())) {
						typeName = string(elem.Type())
					}
					env.SetWithType(name, elem, typeName)
				} else {
					env.Assign(name, elem)
				}
			}
			return val
		}

		// Single-value assignment
		name := node.Names[0]
		if node.Token.Literal == ":=" {
			// nil is untyped, cannot be used with :=
			if val.Type() == NULL_OBJ {
				return &Error{Message: "untyped nil cannot be used with :="}
			}
			// Array value semantics: copy on assignment
			if arr, ok := val.(*Array); ok {
				copied := make([]Object, len(arr.Elements))
				copy(copied, arr.Elements)
				val = &Array{ElemType: arr.ElemType, Elements: copied}
			}
			typeName := ""
			if shouldTrackType(string(val.Type())) {
				typeName = string(val.Type())
			}
			env.SetWithType(name, val, typeName)
		} else {
			// Array value semantics: copy on reassignment too
			if arr, ok := val.(*Array); ok {
				copied := make([]Object, len(arr.Elements))
				copy(copied, arr.Elements)
				val = &Array{ElemType: arr.ElemType, Elements: copied}
			}
			// Fast path: direct lookup in current scope
			if _, hasIt := env.store[name]; hasIt {
				if typeName, hasType := env.types[name]; hasType && typeName != "" {
					// nil can only be assigned to reference types (FUNCTION, ERROR)
					if val.Type() == NULL_OBJ {
						if typeName != string(FUNCTION_OBJ) && typeName != string(ERROR_OBJ) {
							return &Error{Message: fmt.Sprintf("cannot assign nil to variable of type %s", typeName)}
						}
					} else if string(val.Type()) != typeName {
						return &Error{Message: fmt.Sprintf("cannot assign %s to variable of type %s", val.Type(), typeName)}
					}
				}
				env.store[name] = val
				if _, hasType := env.types[name]; !hasType && shouldTrackType(string(val.Type())) {
					env.ensureTypes()
					env.types[name] = string(val.Type())
				}
			} else {
				scope, expectedType, ok := env.ResolveForAssign(name)
				if ok && expectedType != "" {
					// nil can only be assigned to reference types (FUNCTION, ERROR)
					if val.Type() == NULL_OBJ {
						if expectedType != string(FUNCTION_OBJ) && expectedType != string(ERROR_OBJ) {
							return &Error{Message: fmt.Sprintf("cannot assign nil to variable of type %s", expectedType)}
						}
					} else if string(val.Type()) != expectedType {
						return &Error{Message: fmt.Sprintf("cannot assign %s to variable of type %s", val.Type(), expectedType)}
					}
				}
				if scope != nil {
					scope.store[name] = val
					if _, hasType := scope.types[name]; !hasType && shouldTrackType(string(val.Type())) {
						scope.ensureTypes()
						scope.types[name] = string(val.Type())
					}
				} else {
					env.Set(name, val)
				}
			}
		}
		return val
	case *CommandStatement:
		return executeCommand(node.Name, node.Arguments, env, stdin, stdout, stderr)
	case *PrintStatement:
		val := EvalWithIO(node.Expression, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		if i, ok := val.(*Integer); ok {
			fmt.Fprintln(stdout, strconv.FormatInt(i.Value, 10))
		} else if s, ok := val.(*String); ok {
			fmt.Fprintln(stdout, s.Value)
		} else {
			fmt.Fprintln(stdout, inspectObject(val))
		}
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
		if node.Err != "" {
			return &Error{Message: node.Err}
		}
		if node.Obj != nil {
			return node.Obj
		}
		return getIntegerObject(node.Value)
	case *FloatLiteral:
		if node.Err != "" {
			return &Error{Message: node.Err}
		}
		if node.Obj != nil {
			return node.Obj
		}
		return &Float{Value: node.Value}
	case *NilLiteral:
		return NULL
	case *BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)
	case *FunctionLiteral:
		return &Function{Parameters: node.Parameters, ReturnTypes: node.ReturnTypes, Body: node.Body, Env: env}
	}
	return NULL
}

func evalReturnStatement(rs *ReturnStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if len(rs.ReturnValues) == 0 {
		return &ReturnValue{Value: NULL}
	}
	if len(rs.ReturnValues) == 1 {
		val := EvalWithIO(rs.ReturnValues[0], env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}
	}
	// Multi-return: pack into Tuple
	elements := make([]Object, len(rs.ReturnValues))
	for i, rv := range rs.ReturnValues {
		val := EvalWithIO(rv, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		elements[i] = val
	}
	return &ReturnValue{Value: &Tuple{Elements: elements}}
}

func evalArrayLiteral(al *ArrayLiteral, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if len(al.Elements) == 0 {
		return &Array{ElemType: NULL_OBJ, Elements: nil}
	}

	first := EvalWithIO(al.Elements[0], env, stdin, stdout, stderr)
	if isError(first) {
		return first
	}
	elemType := first.Type()

	elements := make([]Object, len(al.Elements))
	elements[0] = first
	for i := 1; i < len(al.Elements); i++ {
		val := EvalWithIO(al.Elements[i], env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		if val.Type() != elemType {
			return &Error{Message: fmt.Sprintf("array type mismatch: expected %s, got %s at index %d", elemType, val.Type(), i)}
		}
		elements[i] = val
	}
	return &Array{ElemType: elemType, Elements: elements}
}

func evalIndexExpression(ie *IndexExpression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	left := EvalWithIO(ie.Left, env, stdin, stdout, stderr)
	if isError(left) {
		return left
	}
	index := EvalWithIO(ie.Index, env, stdin, stdout, stderr)
	if isError(index) {
		return index
	}

	arr, ok := left.(*Array)
	if !ok {
		return &Error{Message: fmt.Sprintf("cannot index non-array type %s", left.Type())}
	}

	idx, ok := index.(*Integer)
	if !ok {
		return &Error{Message: fmt.Sprintf("array index must be INTEGER, got %s", index.Type())}
	}

	i := idx.Value
	if i < 0 || i >= int64(len(arr.Elements)) {
		return &Error{Message: fmt.Sprintf("array index out of bounds: index %d, length %d", i, len(arr.Elements))}
	}
	return arr.Elements[i]
}

func evalStatements(stmts []Statement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var result Object
	for _, statement := range stmts {
		if statement == nil {
			continue
		}
		result = EvalWithIO(statement, env, stdin, stdout, stderr)
		if errObj, ok := result.(*Error); ok {
			env.SetObject("err", errObj)
			return result
		}
		if _, ok := result.(*ReturnValue); ok {
			env.SetObject("err", NULL)
			return result
		}
		if result == BREAK_SIGNAL || result == CONTINUE_SIGNAL {
			return result
		}
	}
	env.SetObject("err", NULL)
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

func evalIterRangeStatement(fs *ForStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	// 1. Evaluate iterator call: iterObj = iter(args)
	iterObj := EvalWithIO(fs.IterCall, env, stdin, stdout, stderr)
	if isError(iterObj) {
		return iterObj
	}

	// 2. Build yield callback: wraps loop body, returns TRUE to continue, FALSE to stop
	var iterResult Object
	yield := &NativeFunction{
		Fn: func(_ *Environment, args ...Object) Object {
			expected := len(fs.IterVars)
			if expected == 0 {
				expected = 1
			}
			if len(args) != expected {
				return &Error{Message: fmt.Sprintf("yield expects %d args, got %d", expected, len(args))}
			}

			// Bind variables
			if len(fs.IterVars) >= 2 && len(args) >= 2 {
				env.SetObject(fs.IterVars[0], args[0])
				env.SetObject(fs.IterVars[1], args[1])
			} else if len(fs.IterVars) >= 1 && len(args) >= 1 {
				env.SetObject(fs.IterVars[0], args[0])
			}

			// Execute loop body
			result := evalLoopBody(fs.Consequence, env, stdin, stdout, stderr)

			if result == BREAK_SIGNAL {
				return FALSE
			}
			if result == CONTINUE_SIGNAL {
				return TRUE
			}
			if isError(result) {
				iterResult = result
				return FALSE
			}
			if _, ok := result.(*ReturnValue); ok {
				iterResult = result
				return FALSE
			}

			return TRUE
		},
	}

	// 3. Call iterator with yield
	switch fn := iterObj.(type) {
	case *NativeFunction:
		ret := fn.Fn(env, yield)
		if isError(ret) {
			return ret
		}
	case *Function:
		ret := applyFunction(fn, []Object{yield}, env, stdin, stdout, stderr)
		if isError(ret) {
			return ret
		}
	default:
		return &Error{Message: fmt.Sprintf("range target is not callable: %s", iterObj.Type())}
	}

	if iterResult != nil {
		return iterResult
	}
	return NULL
}

func evalForStatement(fs *ForStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if fs.IsIterRange {
		return evalIterRangeStatement(fs, env, stdin, stdout, stderr)
	}

	var result Object = NULL
	body := fs.Consequence
	fastCondition, hasFastCondition := buildForConditionFastPath(fs.Condition)

	if fs.HasInc && hasFastCondition && body != nil && len(body.Statements) == 1 && fs.Init == nil && fs.Post == nil {
		return evalForInlinedInc(fs, fastCondition, env, stdin, stdout, stderr)
	}

	if fs.Init != nil {
		result = EvalWithIO(fs.Init, env, stdin, stdout, stderr)
		if isError(result) {
			return result
		}
	}

	for {
		if fs.Condition != nil {
			if hasFastCondition {
				ok, errObj := evalFastForCondition(fastCondition, env)
				if errObj != nil {
					return errObj
				}
				if !ok {
					break
				}
			} else {
				condition := EvalWithIO(fs.Condition, env, stdin, stdout, stderr)
				if isError(condition) {
					return condition
				}
				if !isTruthy(condition) {
					break
				}
			}
		}

		result = evalLoopBody(body, env, stdin, stdout, stderr)
		if result == BREAK_SIGNAL {
			result = NULL
			break
		}
		if result == CONTINUE_SIGNAL {
			if fs.Post != nil {
				postResult := EvalWithIO(fs.Post, env, stdin, stdout, stderr)
				if isError(postResult) {
					return postResult
				}
			}
			continue
		}
		if isError(result) {
			return result
		}
		if _, ok := result.(*ReturnValue); ok {
			return result
		}

		if fs.Post != nil {
			postResult := EvalWithIO(fs.Post, env, stdin, stdout, stderr)
			if isError(postResult) {
				return postResult
			}
		}

		if fs.Condition == nil && fs.Init == nil && fs.Post == nil {
			break
		}
	}
	return result
}

func evalForInlinedInc(fs *ForStatement, cond forConditionFastPath, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var result Object = NULL
	incName := fs.IncVarName
	delta := fs.IncDelta

	for {
		ok, errObj := evalFastForCondition(cond, env)
		if errObj != nil {
			return errObj
		}
		if !ok {
			break
		}

		obj, found := env.GetObject(incName)
		if !found {
			return &Error{Message: "identifier not found: " + incName}
		}
		intObj, ok := obj.(*Integer)
		if !ok {
			return &Error{Message: "type mismatch: for increment requires INTEGER"}
		}
		env.SetObject(incName, getIntegerObject(intObj.Value+delta))
	}
	return result
}

type forConditionFastPath struct {
	varName  string
	op       string
	kind     uint8
	intConst int64
	fltConst float64
	swap     bool
}

const (
	forCondInt uint8 = iota + 1
	forCondFloat
)

func buildForConditionFastPath(expr Expression) (forConditionFastPath, bool) {
	infix, ok := expr.(*InfixExpression)
	if !ok {
		return forConditionFastPath{}, false
	}

	leftIdent, leftIsIdent := infix.Left.(*Identifier)
	rightIdent, rightIsIdent := infix.Right.(*Identifier)

	if leftIsIdent {
		switch right := infix.Right.(type) {
		case *IntegerLiteral:
			return makeForIntFastPath(leftIdent.Value, right.Value, infix.Operator, false)
		case *FloatLiteral:
			return makeForFloatFastPath(leftIdent.Value, right.Value, infix.Operator, false)
		}
	}

	if rightIsIdent {
		switch left := infix.Left.(type) {
		case *IntegerLiteral:
			return makeForIntFastPath(rightIdent.Value, left.Value, infix.Operator, true)
		case *FloatLiteral:
			return makeForFloatFastPath(rightIdent.Value, left.Value, infix.Operator, true)
		}
	}

	return forConditionFastPath{}, false
}

func makeForIntFastPath(name string, constant int64, op string, swap bool) (forConditionFastPath, bool) {
	if !supportsFastCompareOperator(op) {
		return forConditionFastPath{}, false
	}
	return forConditionFastPath{varName: name, intConst: constant, op: op, kind: forCondInt, swap: swap}, true
}

func makeForFloatFastPath(name string, constant float64, op string, swap bool) (forConditionFastPath, bool) {
	if !supportsFastCompareOperator(op) {
		return forConditionFastPath{}, false
	}
	return forConditionFastPath{varName: name, fltConst: constant, op: op, kind: forCondFloat, swap: swap}, true
}

func evalFastForCondition(spec forConditionFastPath, env *Environment) (bool, *Error) {
	obj, ok := env.GetObject(spec.varName)
	if !ok {
		return false, &Error{Message: "identifier not found: " + spec.varName}
	}

	switch spec.kind {
	case forCondInt:
		val, ok := obj.(*Integer)
		if !ok {
			return false, &Error{Message: "type mismatch: " + string(obj.Type()) + " " + spec.op + " INTEGER"}
		}
		left, right := val.Value, spec.intConst
		if spec.swap {
			left, right = right, left
		}
		return compareInts(spec.op, left, right), nil
	case forCondFloat:
		var value float64
		switch v := obj.(type) {
		case *Float:
			value = v.Value
		case *Integer:
			value = float64(v.Value)
		default:
			return false, &Error{Message: "type mismatch: " + string(obj.Type()) + " " + spec.op + " FLOAT"}
		}
		left, right := value, spec.fltConst
		if spec.swap {
			left, right = right, left
		}
		return compareFloats(spec.op, left, right), nil
	default:
		return false, nil
	}
}

func supportsFastCompareOperator(op string) bool {
	switch op {
	case "<", ">", "==", "!=":
		return true
	default:
		return false
	}
}

func compareInts(op string, left, right int64) bool {
	switch op {
	case "<":
		return left < right
	case ">":
		return left > right
	case "==":
		return left == right
	case "!=":
		return left != right
	default:
		return false
	}
}

func compareFloats(op string, left, right float64) bool {
	switch op {
	case "<":
		return left < right
	case ">":
		return left > right
	case "==":
		return left == right
	case "!=":
		return left != right
	default:
		return false
	}
}

func evalLoopBody(body *BlockStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if body == nil {
		return NULL
	}
	var result Object = NULL
	for _, statement := range body.Statements {
		if statement == nil {
			continue
		}
		result = EvalWithIO(statement, env, stdin, stdout, stderr)
		if isError(result) {
			return result
		}
		if _, ok := result.(*ReturnValue); ok {
			return result
		}
		if result == BREAK_SIGNAL || result == CONTINUE_SIGNAL {
			return result
		}
	}
	return result
}

func evalPrefixExpression(node *PrefixExpression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	switch node.Operator {
	case "&":
		// Address-of operator
		if ident, ok := node.Right.(*Identifier); ok {
			ref, ok := env.GetRef(ident.Value)
			if !ok {
				return &Error{Message: "identifier not found: " + ident.Value}
			}
			return &Pointer{Ref: ref, Env: env}
		}
		return &Error{Message: "cannot take address of non-identifier"}
	case "*":
		// Dereference operator
		val := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		ptr, ok := val.(*Pointer)
		if !ok {
			return &Error{Message: "cannot dereference non-pointer"}
		}
		if ptr.Ref == nil {
			return &Error{Message: "nil pointer dereference"}
		}
		return ptr.Ref.Value
	case "!":
		val := EvalWithIO(node.Right, env, stdin, stdout, stderr)
		if isError(val) {
			return val
		}
		return nativeBoolToBooleanObject(!isTruthy(val))
	default:
		return &Error{Message: "unknown prefix operator: " + node.Operator}
	}
}

func evalPointerAssignStatement(node *PointerAssignStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	// The target is *p, we need to evaluate p to get the pointer
	prefixExpr, ok := node.Target.(*PrefixExpression)
	if !ok || prefixExpr.Operator != "*" {
		return &Error{Message: "invalid pointer assignment target"}
	}

	// Evaluate p (the right side of *)
	ptrVal := EvalWithIO(prefixExpr.Right, env, stdin, stdout, stderr)
	if isError(ptrVal) {
		return ptrVal
	}

	ptr, ok := ptrVal.(*Pointer)
	if !ok {
		return &Error{Message: "cannot assign to non-pointer"}
	}

	if ptr.Ref == nil {
		return &Error{Message: "nil pointer dereference"}
	}

	// Evaluate the value
	if node.Value == nil {
		return &Error{Message: "pointer assign: value expression is nil"}
	}
	val := EvalWithIO(node.Value, env, stdin, stdout, stderr)
	if isError(val) {
		return val
	}

	// Assign through pointer - use the pointer's original env
	ptr.Env.SetByPointer(ptr.Ref, val)
	return val
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
	case left.Type() == FLOAT_OBJ && right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == FLOAT_OBJ && right.Type() == INTEGER_OBJ:
		return evalFloatInfixExpression(operator, left, &Float{Value: float64(right.(*Integer).Value)})
	case left.Type() == INTEGER_OBJ && right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, &Float{Value: float64(left.(*Integer).Value)}, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left.(*String).Value, right.(*String).Value)
	case left.Type() == BOOLEAN_OBJ && right.Type() == BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left.(*Boolean).Value, right.(*Boolean).Value)
	case left.Type() == NULL_OBJ && right.Type() == NULL_OBJ:
		return evalNullInfixExpression(operator)
	case left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ:
		return evalArrayInfixExpression(operator, left.(*Array), right.(*Array))
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
	case "-":
		return getIntegerObject(leftVal - rightVal)
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", left.Type(), operator, right.Type())}
	}
}

func evalFloatInfixExpression(operator string, left, right Object) Object {
	leftVal := left.(*Float).Value
	rightVal := right.(*Float).Value

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
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", left.Type(), operator, right.Type())}
	}
}

func evalStringInfixExpression(operator string, left, right string) Object {
	switch operator {
	case "==":
		return nativeBoolToBooleanObject(left == right)
	case "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", STRING_OBJ, operator, STRING_OBJ)}
	}
}

func evalBooleanInfixExpression(operator string, left, right bool) Object {
	switch operator {
	case "==":
		return nativeBoolToBooleanObject(left == right)
	case "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", BOOLEAN_OBJ, operator, BOOLEAN_OBJ)}
	}
}

func evalNullInfixExpression(operator string) Object {
	switch operator {
	case "==":
		return TRUE
	case "!=":
		return FALSE
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", NULL_OBJ, operator, NULL_OBJ)}
	}
}

func evalArrayInfixExpression(operator string, left, right *Array) Object {
	switch operator {
	case "==":
		if left.ElemType != right.ElemType {
			return FALSE
		}
		if len(left.Elements) != len(right.Elements) {
			return FALSE
		}
		for i := range left.Elements {
			eq := evalInfixExpression("==", left.Elements[i], right.Elements[i])
			if eq != TRUE {
				return FALSE
			}
		}
		return TRUE
	case "!=":
		eq := evalArrayInfixExpression("==", left, right)
		if eq == TRUE {
			return FALSE
		}
		return TRUE
	default:
		return &Error{Message: fmt.Sprintf("unknown operator: %s %s %s", ARRAY_OBJ, operator, ARRAY_OBJ)}
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
	if n == 0 {
		return NULL
	}

	if canUseSequentialPipe(ps, env) {
		return evalPipeStatementSequential(ps, env, stdin, stdout, stderr)
	}

	pipes := make([]*io.PipeWriter, n-1)
	readers := make([]*io.PipeReader, n-1)

	for i := 0; i < n-1; i++ {
		readers[i], pipes[i] = io.Pipe()
	}

	var wg sync.WaitGroup
	wg.Add(n)

	var errs []string
	var errMu sync.Mutex

	for i := range n {
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

func canUseSequentialPipe(ps *PipeStatement, env *Environment) bool {
	for idx, cmd := range ps.Commands {
		switch c := cmd.(type) {
		case *PrintStatement:
			continue
		case *CommandStatement:
			if commandRunsInProcess(c.Name, env) {
				continue
			}
			return false
		default:
			if idx == len(ps.Commands)-1 {
				return false
			}
			return false
		}
	}
	return true
}

func commandRunsInProcess(name string, env *Environment) bool {
	if _, ok := builtin.Builtins[name]; ok {
		return true
	}
	if _, ok := NativeFns[name]; ok {
		return true
	}
	if val, ok := env.GetObject(name); ok {
		switch val.(type) {
		case *Function, *NativeFunction:
			return true
		}
	}
	return false
}

func evalPipeStatementSequential(ps *PipeStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	curStdin := stdin
	for idx, cmd := range ps.Commands {
		isLast := idx == len(ps.Commands)-1
		if isLast {
			res := EvalWithIO(cmd, env, curStdin, stdout, stderr)
			if isError(res) {
				return res
			}
			return NULL
		}

		var out bytes.Buffer
		res := EvalWithIO(cmd, env, curStdin, &out, stderr)
		if isError(res) {
			return res
		}
		curStdin = bytes.NewReader(out.Bytes())
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
	// 1. Check for native functions (global)
	if fn, ok := NativeFns[name]; ok {
		evaledArgs, errObj := evalCommandArgsAsObjects(args, env, stdin, stdout, stderr)
		if errObj != nil {
			return errObj
		}
		return fn.Fn(env, evaledArgs...)
	}

	// 2. Check for user-defined functions or native functions in env
	if val, ok := env.GetObject(name); ok {
		if fn, ok := val.(*Function); ok {
			stringArgs, errObj := evalCommandArgsAsStringObjects(args, env, stdin, stdout, stderr)
			if errObj != nil {
				return errObj
			}
			return applyFunction(fn, stringArgs, env, stdin, stdout, stderr)
		}
		if fn, ok := val.(*NativeFunction); ok {
			evaledArgs, errObj := evalCommandArgsAsObjects(args, env, stdin, stdout, stderr)
			if errObj != nil {
				return errObj
			}
			return fn.Fn(env, evaledArgs...)
		}
	}

	// 3. Check for builtins
	if cmd, ok := builtin.Builtins[name]; ok {
		strArgs, errObj := evalCommandArgsAsStrings(args, env, stdin, stdout, stderr)
		if errObj != nil {
			return errObj
		}
		exitCode := cmd.Action(strArgs, env, stdin, stdout, stderr)
		if exitCode != 0 {
			return &Error{Message: fmt.Sprintf("builtin %s failed", name), Code: exitCode, Op: name}
		}
		return NULL
	}

	// 4. External command
	strArgs, errObj := evalCommandArgsAsStrings(args, env, stdin, stdout, stderr)
	if errObj != nil {
		return errObj
	}
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
	if ident, ok := node.Object.(*Identifier); ok {
		if ident.Value == "env" {
			name := "env." + node.Property
			if fn, ok := NativeFns[name]; ok {
				return fn
			}
			return &Error{Message: "unsupported member access: " + name + ", use env.Get() instead"}
		}
		if ident.Value == "param" {
			name := "param." + node.Property
			if fn, ok := NativeFns[name]; ok {
				return fn
			}
			return &Error{Message: "unsupported member access: " + name + ", use param.Get() instead"}
		}
	}

	left := EvalWithIO(node.Object, env, os.Stdin, os.Stdout, os.Stderr)
	if isError(left) {
		return left
	}

	// Handle sync package methods (e.g., sync.NewWaitGroup)
	if pkg, ok := left.(*Package); ok && pkg.Name == "sync" {
		switch node.Property {
		case "NewWaitGroup":
			return &NativeFunction{
				Fn: func(env *Environment, args ...Object) Object {
					if len(args) != 0 {
						return &Error{Message: "NewWaitGroup expects no arguments"}
					}
					return &WaitGroup{Wg: &sync.WaitGroup{}}
				},
			}
		default:
			return &Error{Message: "unknown sync method: " + node.Property}
		}
	}

	// Handle WaitGroup method access (e.g., wg.Wait)
	if wg, ok := left.(*WaitGroup); ok {
		switch node.Property {
		case "Wait":
			return &NativeFunction{
				Fn: func(env *Environment, args ...Object) Object {
					realWg, ok := wg.Wg.(*sync.WaitGroup)
					if !ok {
						return &Error{Message: "invalid WaitGroup"}
					}
					if len(args) > 0 {
						if timeout, ok := args[0].(*Integer); ok {
							done := make(chan struct{})
							go func() { realWg.Wait(); close(done) }()
							select {
							case <-done:
								return NULL
							case <-time.After(time.Duration(timeout.Value) * time.Second):
								return &Error{Message: "WaitGroup timeout"}
							}
						}
					}
					realWg.Wait()
					return NULL
				},
			}
		default:
			return &Error{Message: "unknown method " + node.Property + " on WaitGroup"}
		}
	}

	// Handle Task method access (e.g., t.Wait)
	if task, ok := left.(*Task); ok {
		switch node.Property {
		case "Wait":
			return &NativeFunction{
				Fn: func(env *Environment, args ...Object) Object {
					if len(args) > 0 {
						if timeout, ok := args[0].(*Integer); ok {
							select {
							case <-task.Done:
								if isError(task.Result) {
									return task.Result
								}
								return task.Result
							case <-time.After(time.Duration(timeout.Value) * time.Second):
								return &Error{Message: "Task timeout"}
							}
						}
					}
					<-task.Done
					if isError(task.Result) {
						return task.Result
					}
					return task.Result
				},
			}
		default:
			return &Error{Message: "unknown method " + node.Property + " on Task"}
		}
	}

	name := inspectObject(left) + "." + node.Property
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
		return applyFunction(fn, args, env, stdin, stdout, stderr)
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
			objArgs := make([]Object, len(args))
			for i, arg := range args {
				objArgs[i] = &String{Value: arg}
			}
			return applyFunction(fn, objArgs, env, stdin, stdout, stderr)
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
	if node.Value == "sync" {
		return SYNCPKG
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

func isReturn(obj Object) bool {
	if obj != nil {
		return obj.Type() == RETURN_VALUE_OBJ
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

func evalCommandArgsAsObjects(args []Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) ([]Object, *Error) {
	if len(args) == 0 {
		return nil, nil
	}
	values := make([]Object, len(args))
	for i, arg := range args {
		value := EvalWithIO(arg, env, stdin, stdout, stderr)
		if isError(value) {
			return nil, value.(*Error)
		}
		values[i] = value
	}
	return values, nil
}

func evalCommandArgsAsStrings(args []Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) ([]string, *Error) {
	if len(args) == 0 {
		return nil, nil
	}
	values := make([]string, len(args))
	for i, arg := range args {
		value, errObj := evalCommandArgString(arg, env, stdin, stdout, stderr)
		if errObj != nil {
			return nil, errObj
		}
		values[i] = value
	}
	return values, nil
}

func evalCommandArgsAsStringObjects(args []Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) ([]Object, *Error) {
	if len(args) == 0 {
		return nil, nil
	}
	values := make([]Object, len(args))
	for i, arg := range args {
		value, errObj := evalCommandArgString(arg, env, stdin, stdout, stderr)
		if errObj != nil {
			return nil, errObj
		}
		values[i] = &String{Value: value}
	}
	return values, nil
}

func evalCommandArgString(arg Expression, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) (string, *Error) {
	if literal, ok := arg.(*StringLiteral); ok && strings.IndexByte(literal.Value, '$') < 0 {
		return literal.Value, nil
	}
	value := EvalWithIO(arg, env, stdin, stdout, stderr)
	if isError(value) {
		return "", value.(*Error)
	}
	return inspectObject(value), nil
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

func objectsToStringArgs(args []Object) []Object {
	if len(args) == 0 {
		return nil
	}
	values := make([]Object, len(args))
	for i, arg := range args {
		values[i] = &String{Value: inspectObject(arg)}
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
	case *ReturnValue:
		return inspectObject(v.Value)
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
	env.SetObject(fs.Name, fn)
	return NULL
}

func evalImportStatement(is *ImportStatement, env *Environment) Object {
	path := is.Path

	// 检查是否以"Go"开头
	if !strings.HasPrefix(path, "Go") {
		return &Error{Message: "import path must start with 'Go'"}
	}

	// 移除"Go"前缀
	goPath := strings.TrimPrefix(path, "Go")
	if goPath == "" {
		// import "Go" - 导入整个Go标准库（暂时返回空包）
		env.SetObject("Go", &Package{Name: "Go"})
		return NULL
	}

	// 移除开头的斜杠
	goPath = strings.TrimPrefix(goPath, "/")

	// 检查是否是子包
	if strings.Contains(goPath, "/") {
		// 处理子包，如 "Go/net/http"
		parts := strings.Split(goPath, "/")
		pkgName := parts[len(parts)-1] // 使用最后一个部分作为包名

		// 检查是否有这个包的映射
		if _, ok := goStdlib[pkgName]; ok {
			// 创建包对象
			pkg := &Package{Name: pkgName}
			env.SetObject(pkgName, pkg)

			// 将包中的函数注册到环境中
			for fnName, fn := range goStdlib[pkgName] {
				env.SetObject(pkgName+"."+fnName, fn)
			}
			return NULL
		}
		return &Error{Message: "package not found: " + goPath}
	}

	// 处理直接包，如 "Go/fmt"
	pkgName := goPath

	// 检查是否有这个包的映射
	if _, ok := goStdlib[pkgName]; ok {
		// 创建包对象
		pkg := &Package{Name: pkgName}
		env.SetObject(pkgName, pkg)

		// 将包中的函数注册到环境中
		for fnName, fn := range goStdlib[pkgName] {
			env.SetObject(pkgName+"."+fnName, fn)
		}
		return NULL
	}

	return &Error{Message: "package not found: " + goPath}
}

func applyFunction(fn *Function, args []Object, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	extendEnv := NewFunctionCallEnvironment(fn.Env, len(fn.Parameters))

	for i, param := range fn.Parameters {
		if i < len(args) {
			extendEnv.SetObject(param.Name, args[i])
		}
	}

	result := EvalWithIO(fn.Body, extendEnv, stdin, stdout, stderr)
	if returnValue, ok := result.(*ReturnValue); ok {
		return returnValue.Value
	}
	return result
}

func evalMethodCallBlock(mcb *MethodCallBlockStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	// Evaluate the object
	obj := EvalWithIO(mcb.Object, env, stdin, stdout, stderr)
	if isError(obj) {
		return obj
	}

	switch o := obj.(type) {
	case *WaitGroup:
		switch mcb.Method {
		case "Go":
			wg, ok := o.Wg.(*sync.WaitGroup)
			if !ok {
				return &Error{Message: "invalid WaitGroup"}
			}
			wg.Add(1)
			asyncEnv := env.Clone()
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Fprintf(stderr, "panic in wg.Go: %v\n", r)
					}
					wg.Done()
				}()
				EvalWithIO(mcb.Body, asyncEnv, stdin, stdout, stderr)
			}()
			return NULL
		case "Wait":
			wg, ok := o.Wg.(*sync.WaitGroup)
			if !ok {
				return &Error{Message: "invalid WaitGroup"}
			}
			wg.Wait()
			return NULL
		default:
			return &Error{Message: fmt.Sprintf("unknown method %s on WaitGroup", mcb.Method)}
		}
	default:
		return &Error{Message: fmt.Sprintf("method call with block not supported on %s", obj.Type())}
	}
}

func evalWaitStatement(ws *WaitStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if ws.Timeout != nil {
		timeoutVal := EvalWithIO(ws.Timeout, env, stdin, stdout, stderr)
		if isError(timeoutVal) {
			return timeoutVal
		}
		if timeout, ok := timeoutVal.(*Integer); ok {
			deadline := time.Now().Add(time.Duration(timeout.Value) * time.Second)
			for {
				if allJobsDone() {
					return NULL
				}
				if time.Now().After(deadline) {
					return &Error{Message: "Wait timeout"}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	for {
		if allJobsDone() {
			return NULL
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func allJobsDone() bool {
	builtin.JobsMu.Lock()
	defer builtin.JobsMu.Unlock()
	for _, job := range builtin.Jobs {
		if job.Status == "Running" {
			return false
		}
	}
	return true
}

func evalSwitchStatement(ss *SwitchStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if ss.IntSwitch {
		return evalIntSwitch(ss, env, stdin, stdout, stderr)
	}
	if ss.StringSwitch {
		return evalStringSwitch(ss, env, stdin, stdout, stderr)
	}
	return evalSwitchFallback(ss, env, stdin, stdout, stderr)
}

type intCasePair struct {
	val     int64
	caseIdx int
}

func evalIntSwitch(ss *SwitchStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var tagVal Object
	if ss.Tag != nil {
		tagVal = EvalWithIO(ss.Tag, env, stdin, stdout, stderr)
		if isError(tagVal) {
			return tagVal
		}
	}

	var defaultClause *CaseClause
	if tagVal == nil {
		// tagless int switch: evaluate conditions as bool
		return evalSwitchFallback(ss, env, stdin, stdout, stderr)
	}

	tagInt, ok := tagVal.(*Integer)
	if !ok {
		return evalSwitchFallback(ss, env, stdin, stdout, stderr)
	}

	// Build sorted (value, caseIndex) pairs for binary search
	pairs := make([]intCasePair, 0, len(ss.Cases))
	for i := range ss.Cases {
		c := &ss.Cases[i]
		if c.Values == nil {
			defaultClause = c
			continue
		}
		for _, v := range c.IntConsts {
			pairs = append(pairs, intCasePair{val: v, caseIdx: i})
		}
	}

	// Sort by value for binary search
	sortIntCasePairs(pairs)

	// Binary search for tag value
	target := tagInt.Value
	lo, hi := 0, len(pairs)-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if pairs[mid].val == target {
			return EvalWithIO(ss.Cases[pairs[mid].caseIdx].Body, env, stdin, stdout, stderr)
		}
		if pairs[mid].val < target {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	if defaultClause != nil {
		return EvalWithIO(defaultClause.Body, env, stdin, stdout, stderr)
	}
	return NULL
}

func sortIntCasePairs(p []intCasePair) {
	for i := 1; i < len(p); i++ {
		key := p[i]
		j := i - 1
		for j >= 0 && p[j].val > key.val {
			p[j+1] = p[j]
			j--
		}
		p[j+1] = key
	}
}

func evalStringSwitch(ss *SwitchStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var tagVal Object
	if ss.Tag != nil {
		tagVal = EvalWithIO(ss.Tag, env, stdin, stdout, stderr)
		if isError(tagVal) {
			return tagVal
		}
	}

	var defaultClause *CaseClause
	if tagVal == nil {
		return evalSwitchFallback(ss, env, stdin, stdout, stderr)
	}

	tagStr, ok := tagVal.(*String)
	if !ok {
		return evalSwitchFallback(ss, env, stdin, stdout, stderr)
	}

	for i := range ss.Cases {
		c := &ss.Cases[i]
		if c.Values == nil {
			defaultClause = c
			continue
		}
		if !c.HasConstVals {
			return evalSwitchFallback(ss, env, stdin, stdout, stderr)
		}
		if slices.Contains(c.StringConsts, tagStr.Value) {
			return EvalWithIO(c.Body, env, stdin, stdout, stderr)
		}
	}

	if defaultClause != nil {
		return EvalWithIO(defaultClause.Body, env, stdin, stdout, stderr)
	}
	return NULL
}

func evalSwitchFallback(ss *SwitchStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	var tagVal Object
	if ss.Tag != nil {
		tagVal = EvalWithIO(ss.Tag, env, stdin, stdout, stderr)
		if isError(tagVal) {
			return tagVal
		}
	}

	var defaultClause *CaseClause
	for i := range ss.Cases {
		c := &ss.Cases[i]

		if c.Values == nil {
			defaultClause = c
			continue
		}

		// tagless switch: case condition is a bool expression
		if tagVal == nil {
			for _, v := range c.Values {
				result := EvalWithIO(v, env, stdin, stdout, stderr)
				if isError(result) {
					return result
				}
				if isTruthy(result) {
					return EvalWithIO(c.Body, env, stdin, stdout, stderr)
				}
			}
			continue
		}

		// tagged switch: compare tag == case value
		for _, v := range c.Values {
			caseVal := EvalWithIO(v, env, stdin, stdout, stderr)
			if isError(caseVal) {
				return caseVal
			}
			eq := evalInfixExpression("==", tagVal, caseVal)
			if isError(eq) {
				return eq
			}
			if eq == TRUE {
				return EvalWithIO(c.Body, env, stdin, stdout, stderr)
			}
		}
	}

	if defaultClause != nil {
		return EvalWithIO(defaultClause.Body, env, stdin, stdout, stderr)
	}

	return NULL
}

func evalVarStatement(vs *VarStatement, env *Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) Object {
	if vs == nil {
		return &Error{Message: "invalid var statement"}
	}
	typeName := ""
	if vs.TypeName != "" {
		typeName = string(mapTypeName(vs.TypeName))
	}

	val, errObj := evaluateDeclaredValue(vs, env, stdin, stdout, stderr, typeName)
	if errObj != nil {
		return errObj
	}

	if typeName == "" && shouldTrackType(string(val.Type())) {
		typeName = string(val.Type())
	}

	env.SetWithType(vs.Name, val, typeName)
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

	if typeName != "" {
		// nil can only be assigned to reference types (FUNCTION, ERROR)
		if val.Type() == NULL_OBJ {
			if typeName != string(FUNCTION_OBJ) && typeName != string(ERROR_OBJ) {
				return nil, &Error{Message: fmt.Sprintf("cannot initialize %s with nil", typeName)}
			}
		} else if string(val.Type()) != typeName {
			return nil, &Error{Message: fmt.Sprintf("cannot initialize %s with value of type %s", typeName, val.Type())}
		}
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
	case "func", "function":
		return FUNCTION_OBJ
	case "error":
		return ERROR_OBJ
	case "array":
		return ARRAY_OBJ
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
	case ARRAY_OBJ:
		return &Array{ElemType: NULL_OBJ, Elements: nil}
	default:
		return NULL
	}
}
