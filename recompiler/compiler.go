package recompiler

import (
	"fmt"
	"go/format"
	"os"
	"strconv"
	"strings"

	"kamishell/builtin"
	"kamishell/core"
)

type CompiledScript struct {
	Source  string
	Imports []string
}

type goType string

const (
	goInt    goType = "int64"
	goFloat  goType = "float64"
	goStr    goType = "string"
	goBool   goType = "bool"
	goAny    goType = "any"
	goArrAny goType = "[]any"
)

func (t goType) zero() string {
	switch t {
	case goInt:
		return "0"
	case goFloat:
		return "0.0"
	case goStr:
		return "\"\""
	case goBool:
		return "false"
	}
	return "nil"
}

type compiler struct {
	buf            strings.Builder
	imports        map[string]string
	symbols        map[string]goType
	funcDefs       []string
	knownFuncs     map[string]bool
	funcReturns    map[string]goType   // function name -> first return type (for type inference)
	funcReturnList map[string][]goType // function name -> all return types (for multi-value unpack)
	arrayTypes     map[string]goType
	envSync        map[string]bool
	parentTypes    map[string]goType
	funcLiteralVars map[string]*core.FunctionLiteral // variables assigned function literals
	waitGroups     map[string]bool // variables that are *sync.WaitGroup
	importedPkgs   map[string]bool // imported Go package names (e.g., "fmt", "math")
	loopDepth      int
	indentLv       int
	usesErr        bool
	usesEnv        bool
	funcNeedsEnv   map[string]bool
	err            error
	hasErr         bool
}

func (c *compiler) indent()    { c.indentLv++ }
func (c *compiler) dedent()    { c.indentLv-- }
func (c *compiler) line(f string, a ...any) {
	for range c.indentLv {
		c.buf.WriteString("\t")
	}
	c.buf.WriteString(fmt.Sprintf(f, a...))
	c.buf.WriteString("\n")
}

func (c *compiler) write(f string, a ...any) {
	c.buf.WriteString(fmt.Sprintf(f, a...))
}

func (c *compiler) errorf(format string, a ...any) {
	c.hasErr = true
	c.err = fmt.Errorf(format, a...)
}

func (c *compiler) addImport(pkg, alias string) {
	if c.imports == nil {
		c.imports = make(map[string]string)
	}
	if alias != "" {
		c.imports[pkg] = alias
	} else if _, ok := c.imports[pkg]; !ok {
		c.imports[pkg] = alias
	}
}

func (c *compiler) declareVar(name string, t goType) {
	c.symbols[name] = t
}

func (c *compiler) getVarType(name string) (goType, bool) {
	t, ok := c.symbols[name]
	return t, ok
}

func (c *compiler) hasVar(name string) bool {
	_, ok := c.symbols[name]
	return ok
}

// genEnvSync generates a SetString call to sync a variable to kamiEnv.
// Converts the Go value to string based on its type.
func (c *compiler) genEnvSync(name string) {
	c.usesEnv = true
	varType, ok := c.symbols[name]
	if !ok {
		varType = goAny
	}
	switch varType {
	case goInt:
		c.addImport("strconv", "")
		c.line("kamiEnv.SetString(%q, strconv.FormatInt(%s, 10))", name, name)
	case goFloat:
		c.addImport("strconv", "")
		c.line("kamiEnv.SetString(%q, strconv.FormatFloat(%s, 'f', -1, 64))", name, name)
	case goBool:
		c.addImport("strconv", "")
		c.line("kamiEnv.SetString(%q, strconv.FormatBool(%s))", name, name)
	case goStr:
		c.line("kamiEnv.SetString(%q, %s)", name, name)
	default:
		c.addImport("kamishell/recompiler", "")
		c.line("kamiEnv.SetString(%q, recompiler.ToStr(%s))", name, name)
	}
}

// analyzeEnvDependencies walks the AST and determines which variables need 
// environment synchronization (kamiEnv.Set). Variables need sync if they are:
// 1. Referenced via $var in string interpolation
// 2. Referenced via $var in command arguments (for builtin env.Get access)
// 3. Captured by closures
func analyzeEnvDependencies(program *core.Program) map[string]bool {
	needsSync := make(map[string]bool)

	// First pass: collect all declared variable names
	allDecls := make(map[string]bool)
	var collectDecls func(stmts []core.Statement)
	collectDecls = func(stmts []core.Statement) {
		for _, stmt := range stmts {
			switch s := stmt.(type) {
			case *core.AssignStatement:
				if s.Token.Literal == ":=" {
					for _, name := range s.Names {
						allDecls[name] = true
					}
				}
			case *core.VarStatement:
				allDecls[s.Name] = true
			case *core.ForStatement:
				if s.Init != nil {
					collectDecls([]core.Statement{s.Init})
				}
				if s.Consequence != nil {
					collectDecls(s.Consequence.Statements)
				}
			case *core.BlockStatement:
				collectDecls(s.Statements)
			case *core.IfStatement:
				if s.Consequence != nil {
					collectDecls(s.Consequence.Statements)
				}
				if s.Alternative != nil {
					collectDecls(s.Alternative.Statements)
				}
			}
		}
	}
	collectDecls(program.Statements)

	var walkStatements func(stmts []core.Statement, outerVars map[string]bool)
	var walkExpression func(expr core.Expression, outerVars map[string]bool)

	walkStatements = func(stmts []core.Statement, outerVars map[string]bool) {
		for _, stmt := range stmts {
			if stmt == nil {
				continue
			}
			switch s := stmt.(type) {
			case *core.ExpressionStatement:
				if s.Expression != nil {
					walkExpression(s.Expression, outerVars)
				}
			case *core.PrintStatement:
				if s.Expression != nil {
					walkExpression(s.Expression, outerVars)
				}
			case *core.AssignStatement:
				if s.Value != nil {
					walkExpression(s.Value, outerVars)
				}
				if s.Target != nil {
					walkExpression(s.Target, outerVars)
				}
			case *core.VarStatement:
				if s.Value != nil {
					walkExpression(s.Value, outerVars)
				}
			case *core.IfStatement:
				if s.Condition != nil {
					walkExpression(s.Condition, outerVars)
				}
				if s.Consequence != nil {
					walkStatements(s.Consequence.Statements, outerVars)
				}
				if s.Alternative != nil {
					walkStatements(s.Alternative.Statements, outerVars)
				}
			case *core.ForStatement:
				if s.Init != nil {
					walkStatements([]core.Statement{s.Init}, outerVars)
				}
				if s.Condition != nil {
					walkExpression(s.Condition, outerVars)
				}
				if s.Post != nil {
					walkStatements([]core.Statement{s.Post}, outerVars)
				}
				if s.Consequence != nil {
					walkStatements(s.Consequence.Statements, outerVars)
				}
			case *core.SwitchStatement:
				if s.Tag != nil {
					walkExpression(s.Tag, outerVars)
				}
				for _, c := range s.Cases {
					for _, v := range c.Values {
						walkExpression(v, outerVars)
					}
					if c.Body != nil {
						walkStatements(c.Body.Statements, outerVars)
					}
				}
			case *core.ReturnStatement:
				for _, rv := range s.ReturnValues {
					walkExpression(rv, outerVars)
				}
			case *core.CommandStatement:
				// Command arguments with $var reference — mark those vars as needing sync
				for _, arg := range s.Arguments {
					if sl, ok := arg.(*core.StringLiteral); ok && strings.Contains(sl.Value, "$") {
						// Extract $var names from the string
						os.Expand(sl.Value, func(name string) string {
							needsSync[name] = true
							return ""
						})
					}
					walkExpression(arg, outerVars)
				}
			case *core.PipeStatement:
				for _, cmd := range s.Commands {
					walkStatements([]core.Statement{cmd}, outerVars)
				}
			case *core.RedirectStatement:
				walkStatements([]core.Statement{s.Source}, outerVars)
				if s.Target != nil {
					walkExpression(s.Target, outerVars)
				}
			case *core.LogicalStatement:
				walkStatements([]core.Statement{s.Left}, outerVars)
				walkStatements([]core.Statement{s.Right}, outerVars)
			case *core.FunctionStatement:
				// Create a new outerVars set for the function body
				// All collected declarations are potential outer vars for closures
				funcOuter := make(map[string]bool)
				for k, v := range allDecls {
					funcOuter[k] = v
				}
				// The function name itself is in outer scope
				funcOuter[s.Name] = true
				// Remove parameters from outer vars (they're local)
				for _, p := range s.Parameters {
					delete(funcOuter, p.Name)
				}
				if s.Body != nil {
					walkStatements(s.Body.Statements, funcOuter)
				}
			case *core.GoStatement:
				if block, ok := s.Node.(*core.BlockStatement); ok {
					walkStatements(block.Statements, outerVars)
				}
				if cmd, ok := s.Node.(*core.CommandStatement); ok {
					walkStatements([]core.Statement{cmd}, outerVars)
				}
			case *core.BackgroundStatement:
				walkStatements([]core.Statement{s.Stmt}, outerVars)
			case *core.ExecStatement:
				// exec "echo $msg" — CommandStr may contain $var references
				if s.CommandStr != nil {
					walkExpression(s.CommandStr, outerVars)
				}
			case *core.BlockStatement:
				walkStatements(s.Statements, outerVars)
			}
		}
	}

	walkExpression = func(expr core.Expression, outerVars map[string]bool) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *core.Identifier:
			// Closure capture: if this identifier is defined in an outer scope
			// and we're inside a function literal, it needs env sync
			if outerVars[e.Value] {
				needsSync[e.Value] = true
			}
		case *core.InfixExpression:
			walkExpression(e.Left, outerVars)
			walkExpression(e.Right, outerVars)
		case *core.PrefixExpression:
			walkExpression(e.Right, outerVars)
		case *core.CallExpression:
			walkExpression(e.Function, outerVars)
			for _, arg := range e.Arguments {
				walkExpression(arg, outerVars)
			}
		case *core.MemberExpression:
			walkExpression(e.Object, outerVars)
		case *core.IndexExpression:
			walkExpression(e.Left, outerVars)
			walkExpression(e.Index, outerVars)
		case *core.StringLiteral:
			// Check for $var interpolation
			if strings.Contains(e.Value, "$") {
				os.Expand(e.Value, func(name string) string {
					needsSync[name] = true
					return ""
				})
			}
		case *core.ArrayLiteral:
			for _, el := range e.Elements {
				walkExpression(el, outerVars)
			}
		case *core.FunctionLiteral:
			// Closure: variables from allDecls that are used inside are captured
			closureOuter := make(map[string]bool)
			for k, v := range allDecls {
				closureOuter[k] = v
			}
			// Remove parameters (they're local)
			for _, p := range e.Parameters {
				delete(closureOuter, p.Name)
			}
			if e.Body != nil {
				walkStatements(e.Body.Statements, closureOuter)
			}
		}
	}

	// Second pass: analyze references
	// Top-level variables are LOCAL (not outer), so pass empty outerVars
	// outerVars are only populated when entering function bodies
	walkStatements(program.Statements, make(map[string]bool))

	return needsSync
}

func Compile(program *core.Program) (*CompiledScript, error) {
	// First pass: analyze which variables need environment synchronization
	envSync := analyzeEnvDependencies(program)

	comp := &compiler{
		symbols: make(map[string]goType),
		envSync: envSync,
	}

	// Track whether kamiErr/kamiEnv are actually used
	comp.buf.WriteString("func kami_main() {\n")
	comp.indentLv = 1

	// Placeholder for conditional declarations — replaced after compilation loop
	comp.line("__KAMI_DECL_MARKER__")

	for _, stmt := range program.Statements {
		comp.compileStatement(stmt)
		if comp.hasErr {
			return nil, comp.err
		}
	}

	comp.line("")
	comp.dedent()
	comp.buf.WriteString("}\n")

	// Replace placeholder with actual declarations based on usage analysis
	bodyStr := comp.buf.String()
	kamiErrReferenced := strings.Contains(bodyStr, "kamiErr")

	var decls string
	if comp.usesEnv || comp.usesErr || kamiErrReferenced {
		decls += "\tvar kamiErr error\n"
		if !kamiErrReferenced && !comp.usesErr {
			// kamiErr is declared but never referenced in any expression
			decls += "\t_ = kamiErr\n"
		}
	}
	if comp.usesEnv {
		decls += "\tkamiEnv := recompiler.NewEnv()\n"
		comp.addImport("kamishell/recompiler", "")
	}
	if decls != "" {
		body := strings.ReplaceAll(bodyStr, "\t__KAMI_DECL_MARKER__\n", decls)
		comp.buf.Reset()
		comp.buf.WriteString(body)
	} else {
		// Remove marker line entirely — no declarations needed
		body := strings.ReplaceAll(bodyStr, "\t__KAMI_DECL_MARKER__\n", "")
		comp.buf.Reset()
		comp.buf.WriteString(body)
	}

	// Always need recompiler import: main() calls recompiler.ResetImports()
	comp.addImport("kamishell/recompiler", "")

	funcDefs := strings.Join(comp.funcDefs, "\n")

	var impLines []string
	for pkg, alias := range comp.imports {
		if alias != "" {
			impLines = append(impLines, fmt.Sprintf("\t%s %q", alias, pkg))
		} else {
			impLines = append(impLines, fmt.Sprintf("\t%q", pkg))
		}
	}

	var src strings.Builder
	src.WriteString("package main\n\nimport (\n")
	src.WriteString(strings.Join(impLines, "\n"))
	src.WriteString("\n)\n\n")
	src.WriteString(comp.buf.String())
	if funcDefs != "" {
		src.WriteString("\n")
		src.WriteString(funcDefs)
	}
	src.WriteString("\nfunc main() {\n\trecompiler.ResetImports()\n\tkami_main()\n}\n")

	formatted, err := format.Source([]byte(src.String()))
	if err != nil {
		return &CompiledScript{Source: src.String()}, fmt.Errorf("go fmt: %w\n---\n%s", err, src.String())
	}

	imports := make([]string, 0, len(comp.imports))
	for pkg := range comp.imports {
		imports = append(imports, pkg)
	}

	return &CompiledScript{Source: string(formatted), Imports: imports}, nil
}

func (c *compiler) compileStatement(stmt core.Statement) {
	if stmt == nil || c.hasErr {
		return
	}

	switch s := stmt.(type) {
	case *core.ExpressionStatement:
		c.compileExpressionStatement(s)
	case *core.PrintStatement:
		c.compilePrintStatement(s)
	case *core.AssignStatement:
		c.compileAssignStatement(s)
	case *core.VarStatement:
		c.compileVarStatement(s)
	case *core.IfStatement:
		c.compileIfStatement(s)
	case *core.ForStatement:
		c.compileForStatement(s)
	case *core.SwitchStatement:
		c.compileSwitchStatement(s)
	case *core.BlockStatement:
		c.line("{")
		c.indent()
		for _, st := range s.Statements {
			c.compileStatement(st)
		}
		c.dedent()
		c.line("}")
	case *core.ReturnStatement:
		if len(s.ReturnValues) == 0 {
			c.line("return")
		} else if len(s.ReturnValues) == 1 {
			val := c.compileExpression(s.ReturnValues[0])
			c.line("return %s", val)
		} else {
			vals := make([]string, len(s.ReturnValues))
			for i, rv := range s.ReturnValues {
				vals[i] = c.compileExpression(rv)
			}
			c.line("return %s", strings.Join(vals, ", "))
		}
	case *core.BreakStatement:
		c.line("break")
	case *core.ContinueStatement:
		c.line("continue")
	case *core.CommandStatement:
		c.compileCommandStatement(s)
	case *core.PipeStatement:
		c.compilePipeStatement(s)
	case *core.RedirectStatement:
		c.compileRedirectStatement(s)
	case *core.LogicalStatement:
		c.compileLogicalStatement(s)
	case *core.GoStatement:
		c.compileGoStatement(s)
	case *core.BackgroundStatement:
		c.compileBackgroundStatement(s)
	case *core.WaitStatement:
		c.compileWaitStatement(s)
	case *core.ExecStatement:
		c.compileExecStatement(s)
	case *core.PointerAssignStatement:
		c.compilePointerAssignStatement(s)
	case *core.MethodCallBlockStatement:
		c.compileMethodCallBlockStatement(s)
	case *core.FunctionStatement:
		c.compileFunctionStatement(s)
	case *core.ImportStatement:
		c.compileImportStatement(s)
	case *core.InvalidStatement:
		c.errorf("compile error: %s", s.Message)
	default:
		c.errorf("unknown statement type: %T", stmt)
	}
}

func (c *compiler) compileExpressionStatement(s *core.ExpressionStatement) {
	val := c.compileExpression(s.Expression)
	if val != "" {
		// WaitGroup method calls — check if it's a void or error-returning call
		if ce, ok := s.Expression.(*core.CallExpression); ok {
			if me, ok := ce.Function.(*core.MemberExpression); ok {
				if id, ok := me.Object.(*core.Identifier); ok && c.waitGroups != nil && c.waitGroups[id.Value] {
					if me.Property == "Wait" && len(ce.Arguments) > 0 {
						// wg.Wait(timeout) returns error — assign to kamiErr
						c.line("kamiErr = %s", val)
						c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
						c.addImport("kamishell/recompiler", "")
						return
					}
					// wg.Wait() or wg.Go — void, just call
					c.line("%s", val)
					return
				}
			}
		}
		c.line("_ = %s", val)
	}
}

func (c *compiler) compilePrintStatement(s *core.PrintStatement) {
	c.addImport("kamishell/kamilib", "")
	val := c.compileExpression(s.Expression)
	c.line("kamilib.KamiPrint(%s)", val)
}

func (c *compiler) compileAssignStatement(s *core.AssignStatement) {
	if s.Target != nil {
		// Index assignment: arr[i] = val
		idxExpr, ok := s.Target.(*core.IndexExpression)
		if !ok {
			c.errorf("unexpected target type for index assign: %T", s.Target)
			return
		}
		arr := c.compileExpression(idxExpr.Left)
		idx := c.compileExpression(idxExpr.Index)
		val := c.compileExpression(s.Value)
		// Inline typed array assignment
		if arrType, ok := c.inferArrayType(idxExpr.Left); ok {
			switch arrType {
			case goInt, goStr, goFloat, goBool:
				c.line("%s[%s] = %s", arr, idx, val)
				return
			}
		}
		c.line("%s = recompiler.ArraySet(%s, %s, %s)", arr, arr, idx, val)
		return
	}

	val := c.compileExpression(s.Value)

	// Multi-value assignment: val, err := div(10, 0)
	if len(s.Names) > 1 {
		// Call the function and unpack the tuple
		if ce, ok := s.Value.(*core.CallExpression); ok {
			if id, ok := ce.Function.(*core.Identifier); ok && c.knownFuncs != nil && c.knownFuncs[id.Value] {
				// Direct call to known function - generate multi-return unpacking
				var args []string
				needsEnv := true
				if c.funcNeedsEnv != nil {
					if val, exists := c.funcNeedsEnv[id.Value]; exists {
						needsEnv = val
					}
				}
				if needsEnv {
					c.usesEnv = true
					args = append(args, "kamiEnv")
				}
				for _, a := range ce.Arguments {
					args = append(args, c.compileExpression(a))
				}
				callStr := fmt.Sprintf("kamiFunc_%s(%s)", id.Value, strings.Join(args, ", "))
				// Generate temp vars to capture return values
				tmpVars := make([]string, len(s.Names))
				for i, name := range s.Names {
					tmpVars[i] = fmt.Sprintf("kami_tmp_%s_%d", name, i)
				}
				c.line("%s := %s", strings.Join(tmpVars, ", "), callStr)
				// Assign temp vars to named vars with actual types
				var returnTypes []goType
				if c.funcReturnList != nil {
					if rt, exists := c.funcReturnList[id.Value]; exists {
						returnTypes = rt
					}
				}
				for i, name := range s.Names {
					varType := goAny
					if i < len(returnTypes) {
						varType = returnTypes[i]
					}
					c.declareVar(name, varType)
					c.line("var %s %s = %s", name, varType, tmpVars[i])
					if c.envSync == nil || c.envSync[name] {
						c.genEnvSync(name)
					}
				}
				return
			}
		}
		// Fallback: single value unpacked into multiple names.
		// The function is not known at compile time, so the generated Go
		// function only returns a single value ('any'). Assign the result
		// to the first variable and zero-initialize the rest.
		for i, name := range s.Names {
			c.declareVar(name, goAny)
			if i == 0 {
				c.line("var %s any = %s", name, val)
			} else {
				c.line("var %s any", name)
			}
			if c.envSync == nil || c.envSync[name] {
				c.genEnvSync(name)
			}
		}
		return
	}

	if fl, ok := s.Value.(*core.FunctionLiteral); ok {
		// Generate closure with typed params based on Parameter.TypeName
		var paramTypes []string
		for _, p := range fl.Parameters {
			paramTypes = append(paramTypes, string(c.kamiTypeToGo(p.TypeName)))
		}
		// Use Go type inference — the function literal has a concrete type
		c.declareVar(s.Names[0], goAny)
		// Track as function literal for direct call optimization
		if c.funcLiteralVars == nil {
			c.funcLiteralVars = make(map[string]*core.FunctionLiteral)
		}
		c.funcLiteralVars[s.Names[0]] = fl
		c.line("var %s = %s", s.Names[0], val)
		if c.envSync == nil || c.envSync[s.Names[0]] {
			c.addImport("kamishell/recompiler", "")
			c.line("kamiEnv.SetString(%q, recompiler.ToStr(%s))", s.Names[0], s.Names[0])
		}
		return
	}

	name := s.Names[0]

	if s.Token.Literal == ":=" {
		// Check if this is sync.NewWaitGroup() — use Go type inference
		if ce, ok := s.Value.(*core.CallExpression); ok {
			if me, ok := ce.Function.(*core.MemberExpression); ok {
				if id, ok := me.Object.(*core.Identifier); ok && id.Value == "sync" && me.Property == "NewWaitGroup" {
					c.addImport("sync", "")
					c.declareVar(name, goAny)
					c.line("var %s = &sync.WaitGroup{}", name)
					if c.waitGroups == nil {
						c.waitGroups = make(map[string]bool)
					}
					c.waitGroups[name] = true
					return
				}
			}
		}

		// Variable declaration: infer type from expr
		typ := c.inferGoType(s.Value)
		c.declareVar(name, typ)
		c.line("var %s %s = %s", name, typ, val)

		// Track array element type for typed arrays
		if al, ok := s.Value.(*core.ArrayLiteral); ok && len(al.Elements) > 0 {
			elemType := c.inferGoType(al.Elements[0])
			if c.arrayTypes == nil {
				c.arrayTypes = make(map[string]goType)
			}
			c.arrayTypes[name] = elemType
			// Update variable type to typed array
			switch elemType {
			case goInt:
				c.declareVar(name, "[]int64")
			case goStr:
				c.declareVar(name, "[]string")
			case goFloat:
				c.declareVar(name, "[]float64")
			case goBool:
				c.declareVar(name, "[]bool")
			}
		}
	} else {
		// Reassignment
		if !c.hasVar(name) {
			c.declareVar(name, goAny)
		}
		c.line("%s = %s", name, val)
	}

	// Only sync to env if this variable is referenced via $var or by builtins
	if c.envSync == nil || c.envSync[name] {
		c.genEnvSync(name)
	}
}

func (c *compiler) compileVarStatement(s *core.VarStatement) {
	typ := c.kamiTypeToGo(s.TypeName)
	c.declareVar(s.Name, typ)
	if s.Value != nil {
		val := c.compileExpression(s.Value)
		c.line("var %s %s = %s", s.Name, typ, val)
	} else {
		c.line("var %s %s", s.Name, typ)
	}
	if c.envSync == nil || c.envSync[s.Name] {
		c.genEnvSync(s.Name)
	}
}

func (c *compiler) compileIfStatement(s *core.IfStatement) {
	cond := c.compileExpression(s.Condition)
	// Skip IsTruthy when condition is already a bool expression
	if c.isBoolExpr(s.Condition) {
		c.line("if %s {", cond)
	} else {
		c.addImport("kamishell/recompiler", "")
		c.line("if recompiler.IsTruthy(%s) {", cond)
	}
	c.indent()
	for _, st := range s.Consequence.Statements {
		c.compileStatement(st)
	}
	c.dedent()
	if s.Alternative != nil {
		if c.isSingleIf(s.Alternative) {
			// else if
			innerIf, _ := s.Alternative.Statements[0].(*core.IfStatement)
			c.write("} else ")
			c.compileIfStatement(innerIf)
			return
		}
		c.line("} else {")
		c.indent()
		for _, st := range s.Alternative.Statements {
			c.compileStatement(st)
		}
		c.dedent()
	}
	c.line("}")
}

func (c *compiler) isSingleIf(block *core.BlockStatement) bool {
	if block == nil || len(block.Statements) != 1 {
		return false
	}
	_, ok := block.Statements[0].(*core.IfStatement)
	return ok
}

// isBoolExpr returns true if the expression is guaranteed to produce a bool value.
func (c *compiler) isBoolExpr(expr core.Expression) bool {
	switch e := expr.(type) {
	case *core.BooleanLiteral:
		return true
	case *core.InfixExpression:
		switch e.Operator {
		case "==", "!=", "<", ">", "<=", ">=":
			return true
		}
	case *core.PrefixExpression:
		if e.Operator == "!" {
			return true
		}
	case *core.Identifier:
		if c.isType(e, goBool) {
			return true
		}
	case *core.CallExpression:
		if id, ok := e.Function.(*core.Identifier); ok {
			switch id.Value {
			case "len", "push":
				return false
			}
		}
	}
	return false
}

func (c *compiler) compileForStatement(s *core.ForStatement) {
	if s.IsIterRange {
		c.compileIterRangeStatement(s)
		return
	}

	c.loopDepth++
	if s.Init != nil {
		c.compileStatement(s.Init)
	}
	if s.Condition != nil {
		cond := c.compileExpression(s.Condition)
		var condStr string
		if c.isBoolExpr(s.Condition) {
			condStr = fmt.Sprintf("for %s {", cond)
		} else {
			c.addImport("kamishell/recompiler", "")
			condStr = fmt.Sprintf("for recompiler.IsTruthy(%s) {", cond)
		}
		if s.Post != nil {
			postBuf := c.capturePost(s.Post)
			c.line("%s", condStr)
			c.indent()
			for _, st := range s.Consequence.Statements {
				c.compileStatement(st)
			}
			c.line("%s", postBuf)
			c.dedent()
			c.line("}")
		} else {
			// while-style
			c.line("%s", condStr)
			c.indent()
			for _, st := range s.Consequence.Statements {
				c.compileStatement(st)
			}
			c.dedent()
			c.line("}")
		}
	} else {
		// infinite loop
		if s.Post != nil {
			postBuf := c.capturePost(s.Post)
			c.line("for {")
			c.indent()
			c.line("%s", postBuf)
			c.dedent()
			c.line("}")
		} else {
			c.line("for {")
			c.indent()
			for _, st := range s.Consequence.Statements {
				c.compileStatement(st)
			}
			c.dedent()
			c.line("}")
		}
	}
	c.loopDepth--
}

func (c *compiler) capturePost(s core.Statement) string {
	var buf strings.Builder
	switch a := s.(type) {
	case *core.AssignStatement:
		val := c.compileExpression(a.Value)
		buf.WriteString(fmt.Sprintf("%s = %s", a.Names[0], val))
		if c.hasVar(a.Names[0]) && (c.envSync == nil || c.envSync[a.Names[0]]) {
			varType := c.symbols[a.Names[0]]
			switch varType {
			case goInt:
				buf.WriteString(fmt.Sprintf("; kamiEnv.SetString(%q, strconv.FormatInt(%s, 10))", a.Names[0], a.Names[0]))
			case goStr:
				buf.WriteString(fmt.Sprintf("; kamiEnv.SetString(%q, %s)", a.Names[0], a.Names[0]))
			default:
				buf.WriteString(fmt.Sprintf("; kamiEnv.SetString(%q, recompiler.ToStr(%s))", a.Names[0], a.Names[0]))
			}
		}
	}
	return buf.String()
}

func (c *compiler) compileIterRangeStatement(s *core.ForStatement) {
	iterCall := c.compileExpression(s.IterCall)
	vars := s.IterVars

	if len(vars) == 1 {
		c.line("for %s := range %s {", vars[0], iterCall)
	} else if len(vars) >= 2 {
		idxVar := vars[0]
		valVar := vars[1]
		c.line("for %s, %s := range %s {", idxVar, valVar, iterCall)
	} else {
		c.line("for range %s {", iterCall)
	}
	c.indent()
	for _, st := range s.Consequence.Statements {
		c.compileStatement(st)
	}
	c.dedent()
	c.line("}")
}

func (c *compiler) compileSwitchStatement(s *core.SwitchStatement) {
	if s.Tag != nil {
		tag := c.compileExpression(s.Tag)

		// Check if all case values share a concrete type for native Go switch,
		// avoiding recompiler.ToStr overhead and enabling O(1) dispatch.
		if nativeType, ok := c.canUseNativeSwitch(s.Cases); ok && nativeType != "" {
			c.line("switch %s {", tag)
			c.indent()
			for _, cas := range s.Cases {
				if cas.Values == nil {
					c.line("default:")
				} else {
					var vals []string
					for _, v := range cas.Values {
						vals = append(vals, c.compileExpression(v))
					}
					c.line("case %s:", strings.Join(vals, ", "))
				}
				c.indent()
				for _, st := range cas.Body.Statements {
					c.compileStatement(st)
				}
				c.dedent()
			}
			c.dedent()
			c.line("}")
			return
		}

		// Fallback: string comparison via ToStr
		c.addImport("kamishell/recompiler", "")
		c.line("switch recompiler.ToStr(%s) {", tag)
	} else {
		c.line("switch {")
	}
	c.indent()
	for _, cas := range s.Cases {
		if cas.Values == nil {
			c.line("default:")
		} else {
			var vals []string
			for _, v := range cas.Values {
				vals = append(vals, c.compileExpression(v))
			}
			valStrs := make([]string, len(vals))
			for i, v := range vals {
				valStrs[i] = fmt.Sprintf("recompiler.ToStr(%s)", v)
			}
			c.line("case %s:", strings.Join(valStrs, ", "))
		}
		c.indent()
		for _, st := range cas.Body.Statements {
			c.compileStatement(st)
		}
		c.dedent()
	}
	c.dedent()
	c.line("}")
}

// canUseNativeSwitch checks whether all case values share a concrete type
// (int64, string, bool) that supports a native Go switch, avoiding ToStr overhead.
func (c *compiler) canUseNativeSwitch(cases []core.CaseClause) (goType, bool) {
	var common goType
	for _, cas := range cases {
		if cas.Values == nil {
			continue
		}
		for _, v := range cas.Values {
			vt := c.inferGoType(v)
			if vt != goInt && vt != goStr && vt != goBool {
				return "", false
			}
			if common == "" {
				common = vt
			} else if common != vt {
				return "", false
			}
		}
	}
	return common, common != ""
}

func (c *compiler) compileExpression(expr core.Expression) string {
	if expr == nil || c.hasErr {
		return "nil"
	}

	switch e := expr.(type) {
	case *core.IntegerLiteral:
		return fmt.Sprintf("int64(%d)", e.Value)
	case *core.FloatLiteral:
		return fmt.Sprintf("float64(%v)", e.Value)
	case *core.StringLiteral:
		return c.compileStringLiteral(e)
	case *core.BooleanLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *core.NilLiteral:
		return "nil"
	case *core.Identifier:
		return c.compileIdentifier(e)
	case *core.InfixExpression:
		return c.compileInfixExpression(e)
	case *core.PrefixExpression:
		return c.compilePrefixExpression(e)
	case *core.CallExpression:
		return c.compileCallExpression(e)
	case *core.MemberExpression:
		return c.compileMemberExpression(e)
	case *core.IndexExpression:
		arr := c.compileExpression(e.Left)
		idx := c.compileExpression(e.Index)
		// Inline array access for known typed arrays
		if arrType, ok := c.inferArrayType(e.Left); ok {
			switch arrType {
			case goInt, goStr, goFloat, goBool:
				return fmt.Sprintf("%s[%s]", arr, idx)
			}
		}
		return fmt.Sprintf("recompiler.ArrayGet(%s, %s)", arr, idx)
	case *core.ArrayLiteral:
		return c.compileArrayLiteral(e)
	case *core.FunctionLiteral:
		return c.compileFunctionLiteral(e)
	case *core.GoExpression:
		return c.compileGoExpression(e)
	default:
		c.errorf("unknown expression type: %T", expr)
		return "nil"
	}
}

func (c *compiler) compileStringLiteral(s *core.StringLiteral) string {
	if strings.Contains(s.Value, "$") {
		c.usesEnv = true
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.ExpandStr(%q, kamiEnv)", s.Value)
	}
	// strconv.Quote is used at recompiler-compile-time to produce valid Go source.
	// The generated code contains the quoted string directly, not a strconv.Quote call.
	return strconv.Quote(s.Value)
}

func (c *compiler) compileIdentifier(id *core.Identifier) string {
	name := id.Value
	if name == "nil" {
		return "nil"
	}
	if name == "true" {
		return "true"
	}
	if name == "false" {
		return "false"
	}
	// Check if it's a declared local variable first
	if c.hasVar(name) {
		return name
	}
	// Special: 'err' maps to kamiErr for auto error tracking
	if name == "err" {
		c.usesErr = true
		return "kamiErr"
	}
	// Known user-defined functions: reference the Go function directly
	if c.knownFuncs != nil && c.knownFuncs[name] {
		return fmt.Sprintf("kamiFunc_%s", name)
	}
	// Imported Go package — return package name directly (no env lookup)
	if c.importedPkgs != nil && c.importedPkgs[name] {
		return name
	}
	// Auto-declare from env lookup — use parent types if available for closure capture
	c.usesEnv = true
	c.addImport("kamishell/recompiler", "")
	varType := goAny
	if c.parentTypes != nil {
		if parentType, ok := c.parentTypes[name]; ok {
			varType = parentType
		}
	}
	c.declareVar(name, varType)
	switch varType {
	case goInt:
		c.addImport("strconv", "")
		c.line("var %s int64", name)
		c.line("if v, ok := kamiEnv.GetString(%q); ok { %s, _ = strconv.ParseInt(v, 10, 64) }", name, name)
	case goStr:
		c.line("var %s string", name)
		c.line("if v, ok := kamiEnv.GetString(%q); ok { %s = v }", name, name)
	case goFloat:
		c.addImport("strconv", "")
		c.line("var %s float64", name)
		c.line("if v, ok := kamiEnv.GetString(%q); ok { %s, _ = strconv.ParseFloat(v, 64) }", name, name)
	case goBool:
		c.addImport("strconv", "")
		c.line("var %s bool", name)
		c.line("if v, ok := kamiEnv.GetString(%q); ok { %s, _ = strconv.ParseBool(v) }", name, name)
	default:
		c.line("var %s any", name)
		c.line("if v, ok := kamiEnv.GetString(%q); ok { %s = v }", name, name)
	}
	return name
}

func (c *compiler) compileInfixExpression(e *core.InfixExpression) string {
	left := c.compileExpression(e.Left)
	right := c.compileExpression(e.Right)

	bothInt := c.isType(e.Left, goInt) && c.isType(e.Right, goInt)
	bothStr := c.isType(e.Left, goStr) && c.isType(e.Right, goStr)
	bothFloat := c.isType(e.Left, goFloat) && c.isType(e.Right, goFloat)
	bothBool := c.isType(e.Left, goBool) && c.isType(e.Right, goBool)

	switch e.Operator {
	case "+":
		if bothInt || bothStr {
			return fmt.Sprintf("(%s + %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Add(%s, %s)", left, right)
	case "-":
		if bothInt {
			return fmt.Sprintf("(%s - %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Sub(%s, %s)", left, right)
	case "*":
		if bothInt {
			return fmt.Sprintf("(%s * %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Mul(%s, %s)", left, right)
	case "/":
		if bothInt {
			return fmt.Sprintf("(%s / %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Div(%s, %s)", left, right)
	case "==":
		if bothInt || bothStr || bothFloat || bothBool {
			return fmt.Sprintf("(%s == %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Eq(%s, %s)", left, right)
	case "!=":
		if bothInt || bothStr || bothFloat || bothBool {
			return fmt.Sprintf("(%s != %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.NotEq(%s, %s)", left, right)
	case "<":
		if bothInt || bothFloat {
			return fmt.Sprintf("(%s < %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.LessThan(%s, %s)", left, right)
	case ">":
		if bothInt || bothFloat {
			return fmt.Sprintf("(%s > %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.GreaterThan(%s, %s)", left, right)
	case "<=":
		if bothInt || bothFloat {
			return fmt.Sprintf("(%s <= %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.LessEq(%s, %s)", left, right)
	case ">=":
		if bothInt || bothFloat {
			return fmt.Sprintf("(%s >= %s)", left, right)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.GreaterEq(%s, %s)", left, right)
	default:
		c.errorf("unknown infix operator: %s", e.Operator)
		return "nil"
	}
}

func (c *compiler) compilePrefixExpression(e *core.PrefixExpression) string {
	right := c.compileExpression(e.Right)
	switch e.Operator {
	case "!":
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("!recompiler.IsTruthy(%s)", right)
	case "-":
		return fmt.Sprintf("(-%s)", right)
	case "&":
		// Address-of: create a Ptr with typed getter/setter closures
		if ident, ok := e.Right.(*core.Identifier); ok {
			c.addImport("kamishell/recompiler", "")
			name := ident.Value
			varType := goAny
			if t, ok := c.getVarType(name); ok {
				varType = t
			}
			switch varType {
			case goInt:
				return fmt.Sprintf("recompiler.NewPtrInt64(func() int64 { return %s }, func(v int64) { %s = v })", name, name)
			case goStr:
				return fmt.Sprintf("recompiler.NewPtrString(func() string { return %s }, func(v string) { %s = v })", name, name)
			case goFloat:
				return fmt.Sprintf("recompiler.NewPtrFloat64(func() float64 { return %s }, func(v float64) { %s = v })", name, name)
			case goBool:
				return fmt.Sprintf("recompiler.NewPtrBool(func() bool { return %s }, func(v bool) { %s = v })", name, name)
			default:
				return fmt.Sprintf("recompiler.NewPtr(func() any { return %s }, func(v any) { %s = v })", name, name)
			}
		}
		return fmt.Sprintf("recompiler.NewPtr(func() any { return %s }, func(v any) {})", right)
	case "*":
		// Dereference: call Deref on the Ptr
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.Deref(%s)", right)
	default:
		c.errorf("unknown prefix operator: %s", e.Operator)
		return "nil"
	}
}

func (c *compiler) compileCallExpression(e *core.CallExpression) string {
	// Check if function is an Identifier
	if id, ok := e.Function.(*core.Identifier); ok {
		name := id.Value

		// Builtin native functions
		switch name {
		case "len":
			if len(e.Arguments) > 0 {
				c.addImport("kamishell/recompiler", "")
				arg := c.compileExpression(e.Arguments[0])
				return fmt.Sprintf("recompiler.ArrayLen(%s)", arg)
			}
		case "push":
			if len(e.Arguments) >= 2 {
				c.addImport("kamishell/recompiler", "")
				arr := c.compileExpression(e.Arguments[0])
				val := c.compileExpression(e.Arguments[1])
				return fmt.Sprintf("recompiler.ArrayPush(%s, %s)", arr, val)
			}
		case "error":
			c.addImport("kamishell/recompiler", "")
			if len(e.Arguments) > 0 {
				arg := c.compileExpression(e.Arguments[0])
				return fmt.Sprintf("recompiler.NewError(recompiler.ToStr(%s))", arg)
			}
			return "recompiler.NewError(\"\")"
		}

		// Known user-defined functions: call directly
		if c.knownFuncs != nil && c.knownFuncs[name] {
			c.addImport("kamishell/recompiler", "")
			var args []string
			needsEnv := true
			if c.funcNeedsEnv != nil {
				if val, exists := c.funcNeedsEnv[name]; exists {
					needsEnv = val
				}
			}
			if needsEnv {
				c.usesEnv = true
				args = append(args, "kamiEnv")
			}
			for _, a := range e.Arguments {
				args = append(args, c.compileExpression(a))
			}
			return fmt.Sprintf("kamiFunc_%s(%s)", name, strings.Join(args, ", "))
		}
	}

	// Check if this is a call to a function literal variable — generate direct call
	if id, ok := e.Function.(*core.Identifier); ok && c.funcLiteralVars != nil {
		if _, exists := c.funcLiteralVars[id.Value]; exists {
			// Generate direct call with typed arguments
			var args []string
			for _, a := range e.Arguments {
				args = append(args, c.compileExpression(a))
			}
			return fmt.Sprintf("%s(%s)", id.Value, strings.Join(args, ", "))
		}
	}

	// Check if it's sync.NewWaitGroup() — generate &sync.WaitGroup{}
	if me, ok := e.Function.(*core.MemberExpression); ok {
		if id, ok := me.Object.(*core.Identifier); ok && id.Value == "sync" && me.Property == "NewWaitGroup" {
			c.addImport("sync", "")
			return "&sync.WaitGroup{}"
		}
	}

	// Check if it's wg.Wait() or wg.Wait(timeout) where wg is a known WaitGroup
	if me, ok := e.Function.(*core.MemberExpression); ok {
		if id, ok := me.Object.(*core.Identifier); ok && c.waitGroups != nil && c.waitGroups[id.Value] {
			if me.Property == "Wait" {
				if len(e.Arguments) == 0 {
					return fmt.Sprintf("%s.Wait()", id.Value)
				}
				// wg.Wait(timeout) → kamilib.WaitTimeout(wg, timeout)
				timeout := c.compileExpression(e.Arguments[0])
				c.addImport("kamishell/kamilib", "")
				c.usesErr = true
				return fmt.Sprintf("kamilib.WaitTimeout(%s, %s)", id.Value, timeout)
			}
		}
	}

	// Check if it's a call to an imported Go package function — generate direct call
	if me, ok := e.Function.(*core.MemberExpression); ok {
		if id, ok := me.Object.(*core.Identifier); ok && c.importedPkgs != nil && c.importedPkgs[id.Value] {
			var args []string
			for _, a := range e.Arguments {
				args = append(args, c.compileExpression(a))
			}
			return fmt.Sprintf("%s.%s(%s)", id.Value, me.Property, strings.Join(args, ", "))
		}
	}

	// Generic call: function loaded from env as any
	c.addImport("kamishell/recompiler", "")
	fn := c.compileExpression(e.Function)
	var args []string
	for _, a := range e.Arguments {
		args = append(args, c.compileExpression(a))
	}
	c.usesEnv = true
	return fmt.Sprintf("recompiler.CallFunc(%s, kamiEnv, %s)", fn, strings.Join(args, ", "))
}

func (c *compiler) compileMemberExpression(e *core.MemberExpression) string {
	prop := e.Property

	if id, ok := e.Object.(*core.Identifier); ok {
		// Imported Go package — generate direct package function reference
		if c.importedPkgs != nil && c.importedPkgs[id.Value] {
			return fmt.Sprintf("%s.%s", id.Value, prop)
		}
		switch id.Value {
		case "env":
			c.usesEnv = true
			return fmt.Sprintf("kamiEnv.GetStr(%q)", prop)
		case "param":
			return fmt.Sprintf("recompiler.ToStr(paramGet(%q))", prop)
		}
	}

	// Fallback — should not be reached for well-formed programs
	obj := c.compileExpression(e.Object)
	c.addImport("kamishell/recompiler", "")
	return fmt.Sprintf("recompiler.MemberGet(%s, %q)", obj, prop)
}

func (c *compiler) compileArrayLiteral(a *core.ArrayLiteral) string {
	if len(a.Elements) == 0 {
		return "[]any{}"
	}
	// Infer element type from first element
	elemType := c.inferGoType(a.Elements[0])
	allSameType := true
	for _, el := range a.Elements[1:] {
		if c.inferGoType(el) != elemType {
			allSameType = false
			break
		}
	}

	var elems []string
	for _, el := range a.Elements {
		elems = append(elems, c.compileExpression(el))
	}

	// Generate typed array if all elements are the same type
	if allSameType && elemType == goInt {
		return fmt.Sprintf("[]int64{%s}", strings.Join(elems, ", "))
	}
	if allSameType && elemType == goStr {
		return fmt.Sprintf("[]string{%s}", strings.Join(elems, ", "))
	}
	if allSameType && elemType == goFloat {
		return fmt.Sprintf("[]float64{%s}", strings.Join(elems, ", "))
	}
	if allSameType && elemType == goBool {
		return fmt.Sprintf("[]bool{%s}", strings.Join(elems, ", "))
	}
	return fmt.Sprintf("[]any{%s}", strings.Join(elems, ", "))
}

func (c *compiler) compileFunctionLiteral(f *core.FunctionLiteral) string {
	// Build closure body with sub-compiler that inherits all context
	sub := &compiler{
		symbols:        make(map[string]goType),
		imports:        c.imports,
		knownFuncs:     c.knownFuncs,
		funcReturns:    c.funcReturns,
		funcReturnList: c.funcReturnList,
		arrayTypes:     copyMap(c.arrayTypes),
		envSync:        c.envSync,
		loopDepth:      c.loopDepth,
		funcNeedsEnv:   c.funcNeedsEnv,
		waitGroups:     c.waitGroups,
		importedPkgs:   c.importedPkgs,
	}

	// Register parameters with their declared types
	for _, p := range f.Parameters {
		sub.declareVar(p.Name, c.kamiTypeToGo(p.TypeName))
	}

	// Compile body
	for _, st := range f.Body.Statements {
		sub.compileStatement(st)
	}

	// Generate inline function literal with typed parameters
	var inlineParams []string
	for _, p := range f.Parameters {
		inlineParams = append(inlineParams, fmt.Sprintf("%s %s", p.Name, c.kamiTypeToGo(p.TypeName)))
	}
	inlineParamStr := strings.Join(inlineParams, ", ")

	// Determine return type from declaration
	returnType := "any"
	if len(f.ReturnTypes) == 1 {
		returnType = string(c.kamiTypeToGo(f.ReturnTypes[0]))
	}

	var fd strings.Builder
	fd.WriteString(fmt.Sprintf("func(%s) %s {\n", inlineParamStr, returnType))
	fd.WriteString(sub.buf.String())

	// Add default return if body doesn't end with return
	bodyStr := sub.buf.String()
	lines := strings.Split(strings.TrimRight(bodyStr, "\n"), "\n")
	hasTrailingReturn := false
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if strings.HasPrefix(lastLine, "return ") || lastLine == "return" {
			hasTrailingReturn = true
		}
	}
	if !hasTrailingReturn {
		fd.WriteString(fmt.Sprintf("return %s\n", c.kamiTypeToGo(f.ReturnTypes[0]).zero()))
	}
	fd.WriteString("}")

	return fd.String()
}

func (c *compiler) compileCommandStatement(s *core.CommandStatement) {
	c.usesEnv = true
	name := s.Name

	// Merge adjacent tokens that form flags: "-la" → single arg, "--verbose" → single arg
	mergedArgs := c.mergeCommandArgs(s.Arguments)

	// Build string args
	var strArgs []string
	for _, arg := range mergedArgs {
		strArgs = append(strArgs, c.evalCommandArg(arg))
	}
	argStr := fmt.Sprintf("[]string{%s}", strings.Join(strArgs, ", "))

	// Check if it's a builtin
	if cmd, ok := builtin.Builtins[name]; ok {
		c.addImport("kamishell/builtin", "")
		c.addImport("os", "")
		c.addImport("fmt", "")
		funcName := c.builtinFuncName(cmd.Name)
		c.line("{")
		c.indent()
		c.line("kamiErr = nil")
		c.line("exitCode := builtin.%s(%s, kamiEnv, os.Stdin, os.Stdout, os.Stderr)", funcName, argStr)
		c.line("if exitCode != 0 {")
		c.indent()
		c.line("kamiErr = fmt.Errorf(\"%%s exited with code %%d\", %q, exitCode)", name)
		c.dedent()
		c.line("}")
		c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
		c.dedent()
		c.line("}")
		return
	}

	// External command
	c.addImport("os/exec", "")
	c.addImport("os", "")
	c.addImport("fmt", "")
	c.line("{")
	c.indent()
	c.line("kamiErr = nil")
	c.line("cmd := exec.Command(%q, %s...)", name, argStr)
	c.line("cmd.Stdin = os.Stdin")
	c.line("cmd.Stdout = os.Stdout")
	c.line("cmd.Stderr = os.Stderr")
	c.line("if err := cmd.Run(); err != nil {")
	c.indent()
	c.line("kamiErr = err")
	c.dedent()
	c.line("}")
	c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
	c.dedent()
	c.line("}")
}

// mergeCommandArgs merges adjacent StringLiterals that form flags.
// The lexer splits "-la" into "-" and "la", and "--verbose" into "-", "-", "verbose".
// This function merges them back into single StringLiteral args.
func (c *compiler) mergeCommandArgs(args []core.Expression) []core.Expression {
	if len(args) <= 1 {
		return args
	}
	var merged []core.Expression
	i := 0
	for i < len(args) {
		arg := args[i]
		sl, ok := arg.(*core.StringLiteral)
		if !ok || sl.Value != "-" {
			merged = append(merged, arg)
			i++
			continue
		}
		// Collect consecutive "-" tokens and the final IDENT
		parts := []string{sl.Value}
		j := i + 1
		for j < len(args) {
			next, ok := args[j].(*core.StringLiteral)
			if !ok {
				break
			}
			if next.Value == "-" {
				parts = append(parts, "-")
				j++
				continue
			}
			// Non-dash string: merge with the dashes
			parts = append(parts, next.Value)
			j++
			break
		}
		if j > i+1 {
			// Merged
			merged = append(merged, &core.StringLiteral{Value: strings.Join(parts, "")})
			i = j
		} else {
			merged = append(merged, arg)
			i++
		}
	}
	return merged
}

func (c *compiler) evalCommandArg(arg core.Expression) string {
	switch a := arg.(type) {
	case *core.StringLiteral:
		if strings.Contains(a.Value, "$") {
			c.addImport("kamishell/recompiler", "")
			return fmt.Sprintf("recompiler.ExpandStr(%q, kamiEnv)", a.Value)
		}
		return strconv.Quote(a.Value)
	case *core.IntegerLiteral:
		c.addImport("strconv", "")
		return fmt.Sprintf("strconv.FormatInt(int64(%d), 10)", a.Value)
	case *core.BooleanLiteral:
		if a.Value {
			return `"true"`
		}
		return `"false"`
	case *core.PrefixExpression:
		// In command context, -flag is a string literal: -la, --verbose, etc.
		if a.Operator == "-" {
			right := c.commandArgToString(a.Right)
			return strconv.Quote("-" + right)
		}
		// Other prefix ops (like & or *) - treat as expression
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.ToStr(%s)", c.compileExpression(a))
	case *core.Identifier:
		// Direct strconv for known types
		if c.isType(a, goStr) {
			return a.Value
		}
		if c.isType(a, goInt) {
			c.addImport("strconv", "")
			return fmt.Sprintf("strconv.FormatInt(%s, 10)", a.Value)
		}
		if c.isType(a, goBool) {
			c.addImport("strconv", "")
			return fmt.Sprintf("strconv.FormatBool(%s)", a.Value)
		}
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.ToStr(%s)", a.Value)
	default:
		// For complex expressions, check if we can infer the type
		if c.isType(arg, goStr) {
			return c.compileExpression(arg)
		}
		if c.isType(arg, goInt) {
			c.addImport("strconv", "")
			return fmt.Sprintf("strconv.FormatInt(%s, 10)", c.compileExpression(arg))
		}
		c.addImport("kamishell/recompiler", "")
		val := c.compileExpression(a)
		return fmt.Sprintf("recompiler.ToStr(%s)", val)
	}
}

// commandArgToString extracts a string representation from an expression
// used as a command argument (for building flag strings like -la, --verbose).
func (c *compiler) commandArgToString(expr core.Expression) string {
	switch e := expr.(type) {
	case *core.Identifier:
		return e.Value
	case *core.StringLiteral:
		return e.Value
	case *core.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value)
	default:
		return c.compileExpression(expr)
	}
}

func (c *compiler) compilePipeStatement(s *core.PipeStatement) {
	c.usesEnv = true
	c.addImport("io", "")
	c.addImport("sync", "")

	cmds := s.Commands
	n := len(cmds)
	if n == 0 {
		return
	}
	if n == 1 {
		if cmd, ok := cmds[0].(*core.CommandStatement); ok {
			c.compileCommandStatement(cmd)
		}
		return
	}

	c.line("{")
	c.indent()
	c.line("var wg sync.WaitGroup")

	// Create pipes between commands
	var readers []string
	for range n - 1 {
		rd := fmt.Sprintf("pr_%d", len(readers))
		pw := fmt.Sprintf("pw_%d", len(readers))
		c.line("%s, %s := io.Pipe()", rd, pw)
		readers = append(readers, rd)
	}

	for i, stmt := range cmds {
		cmd, ok := stmt.(*core.CommandStatement)
		if !ok {
			continue
		}

		mergedArgs := c.mergeCommandArgs(cmd.Arguments)
		var strArgs []string
		for _, arg := range mergedArgs {
			strArgs = append(strArgs, c.evalCommandArg(arg))
		}
		argStr := fmt.Sprintf("[]string{%s}", strings.Join(strArgs, ", "))

		wgID := i
		c.line("wg.Add(1)")
		c.line("go func() {")
		c.indent()
		c.line("defer wg.Done()")

		// Wire stdin
		if i == 0 {
			c.line("// first command: stdin from os.Stdin")
		} else {
			c.line("defer func() { _ = pw_%d.Close() }()", wgID-1)
		}

		// Wire stdout
		var stdout string
		if i == n-1 {
			stdout = "os.Stdout"
		} else {
			stdout = fmt.Sprintf("pw_%d", wgID)
			c.line("defer func() { _ = %s.Close() }()", stdout)
		}

		var stdin string
		if i == 0 {
			stdin = "os.Stdin"
		} else {
			stdin = fmt.Sprintf("pr_%d", wgID-1)
		}

		if bcmd, ok := builtin.Builtins[cmd.Name]; ok {
			c.addImport("kamishell/builtin", "")
			c.addImport("os", "")
			funcName := c.builtinFuncName(bcmd.Name)
			c.line("builtin.%s(%s, kamiEnv, %s, %s, os.Stderr)", funcName, argStr, stdin, stdout)
		} else {
			c.addImport("os/exec", "")
			c.addImport("os", "")
			c.line("cmd := exec.Command(%q, %s...)", cmd.Name, argStr)
			c.line("cmd.Stdin = %s", stdin)
			c.line("cmd.Stdout = %s", stdout)
			c.line("cmd.Stderr = os.Stderr")
			c.line("_ = cmd.Run()")
		}

		c.dedent()
		c.line("}()")
	}

	c.line("wg.Wait()")
	c.dedent()
	c.line("}")
}

func (c *compiler) compileRedirectStatement(s *core.RedirectStatement) {
	c.usesEnv = true
	c.addImport("os", "")
	c.addImport("kamishell/recompiler", "")
	c.addImport("kamishell/builtin", "")
	c.addImport("os/exec", "")
	target := c.compileExpression(s.Target)

	var flag string
	if s.Append {
		flag = "os.O_APPEND"
	} else {
		flag = "os.O_TRUNC"
	}

	c.line("{")
	c.indent()
	c.line("f, err := os.OpenFile(recompiler.ToStr(%s), os.O_CREATE|os.O_WRONLY|%s, 0644)", target, flag)
	c.line("if err != nil {")
	c.indent()
	c.line("kamiErr = err")
	c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
	c.line("return")
	c.dedent()
	c.line("}")
	c.line("defer f.Close()")

	// Compile the source statement with redirected output
	if cmd, ok := s.Source.(*core.CommandStatement); ok {
		mergedArgs := c.mergeCommandArgs(cmd.Arguments)
		var strArgs []string
		for _, arg := range mergedArgs {
			strArgs = append(strArgs, c.evalCommandArg(arg))
		}
		argStr := fmt.Sprintf("[]string{%s}", strings.Join(strArgs, ", "))

		if bcmd, ok := builtin.Builtins[cmd.Name]; ok {
			c.addImport("kamishell/builtin", "")
			c.addImport("os", "")
			funcName := c.builtinFuncName(bcmd.Name)
			c.line("kamiErr = nil")
			c.line("builtin.%s(%s, kamiEnv, os.Stdin, f, os.Stderr)", funcName, argStr)
		} else {
			c.addImport("os/exec", "")
			c.addImport("os", "")
			c.line("cmd := exec.Command(%q, %s...)", cmd.Name, argStr)
			c.line("cmd.Stdout = f")
			c.line("cmd.Stderr = os.Stderr")
			c.line("kamiErr = cmd.Run()")
		}
	} else if ps, ok := s.Source.(*core.PipeStatement); ok {
		c.compilePipeWithOutput(ps, "f")
	}

	c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
	c.dedent()
	c.line("}")
}

func (c *compiler) compilePipeWithOutput(ps *core.PipeStatement, output string) {
	c.usesEnv = true
	cmds := ps.Commands
	n := len(cmds)
	if n <= 1 {
		return
	}

	c.addImport("io", "")
	c.addImport("sync", "")
	c.line("var wg sync.WaitGroup")

	for i := range n - 1 {
		c.line("pr_%d, pw_%d := io.Pipe()", i, i)
	}

	for i, stmt := range cmds {
		cmd, ok := stmt.(*core.CommandStatement)
		if !ok {
			continue
		}
		mergedArgs := c.mergeCommandArgs(cmd.Arguments)
		var strArgs []string
		for _, arg := range mergedArgs {
			strArgs = append(strArgs, c.evalCommandArg(arg))
		}
		argStr := fmt.Sprintf("[]string{%s}", strings.Join(strArgs, ", "))

		c.line("wg.Add(1)")
		c.line("go func() {")
		c.indent()
		c.line("defer wg.Done()")

		var stdin, stdout string
		if i == 0 {
			stdin = "os.Stdin"
		} else {
			stdin = fmt.Sprintf("pr_%d", i-1)
		}
		if i == n-1 {
			stdout = output
		} else {
			stdout = fmt.Sprintf("pw_%d", i)
		}
		if i < n-1 {
			c.line("defer %s.Close()", stdout)
		}

		if bcmd, ok := builtin.Builtins[cmd.Name]; ok {
			c.addImport("kamishell/builtin", "")
			c.addImport("os", "")
			funcName := c.builtinFuncName(bcmd.Name)
			c.line("builtin.%s(%s, kamiEnv, %s, %s, os.Stderr)", funcName, argStr, stdin, stdout)
		} else {
			c.addImport("os/exec", "")
			c.addImport("os", "")
			c.line("ecmd := exec.Command(%q, %s...)", cmd.Name, argStr)
			c.line("ecmd.Stdin = %s", stdin)
			c.line("ecmd.Stdout = %s", stdout)
			c.line("ecmd.Stderr = os.Stderr")
			c.line("_ = ecmd.Run()")
		}
		c.dedent()
		c.line("}()")
	}
	c.line("wg.Wait()")
}

func (c *compiler) compileLogicalStatement(s *core.LogicalStatement) {
	if s.Operator == "&&" {
		c.line("{")
		c.indent()
		c.line("kamiErr = nil")
		c.compileStatement(s.Left)
		c.line("if kamiErr == nil {")
		c.indent()
		c.compileStatement(s.Right)
		c.dedent()
		c.line("}")
		c.dedent()
		c.line("}")
	} else if s.Operator == "||" {
		c.line("{")
		c.indent()
		c.line("kamiErr = nil")
		c.compileStatement(s.Left)
		c.line("if kamiErr != nil {")
		c.indent()
		c.compileStatement(s.Right)
		c.dedent()
		c.line("}")
		c.dedent()
		c.line("}")
	}
}

func (c *compiler) compileGoStatement(s *core.GoStatement) {
	c.addImport("kamishell/recompiler", "")
	c.line("{")
	c.indent()
	c.line("id := recompiler.RegisterGoJob(%q)", s.String())
	c.line("go func() {")
	c.indent()
	c.line("defer recompiler.CompleteGoJob(id)")

	if block, ok := s.Node.(*core.BlockStatement); ok {
		for _, st := range block.Statements {
			c.compileStatement(st)
		}
	} else if cmd, ok := s.Node.(*core.CommandStatement); ok {
		// Re-use command statement in goroutine
		c.compileCommandStatementInline(cmd)
	}

	c.dedent()
	c.line("}()")
	c.dedent()
	c.line("}")
}

func (c *compiler) compileBackgroundStatement(s *core.BackgroundStatement) {
	c.addImport("kamishell/recompiler", "")
	c.line("{")
	c.indent()
	c.line("id := recompiler.RegisterGoJob(%q)", s.String())
	c.line("go func() {")
	c.indent()
	c.line("defer recompiler.CompleteGoJob(id)")

	if cmd, ok := s.Stmt.(*core.CommandStatement); ok {
		c.compileCommandStatementInline(cmd)
	} else if block, ok := s.Stmt.(*core.BlockStatement); ok {
		for _, st := range block.Statements {
			c.compileStatement(st)
		}
	} else {
		c.compileStatement(s.Stmt)
	}

	c.dedent()
	c.line("}()")
	c.dedent()
	c.line("}")
}

func (c *compiler) compileCommandStatementInline(s *core.CommandStatement) {
	c.addImport("os", "")
	c.addImport("os/exec", "")
	c.addImport("kamishell/builtin", "")
	var strArgs []string
	for _, arg := range s.Arguments {
		strArgs = append(strArgs, c.evalCommandArg(arg))
	}
	argStr := fmt.Sprintf("[]string{%s}", strings.Join(strArgs, ", "))

	if cmd, ok := builtin.Builtins[s.Name]; ok {
		funcName := c.builtinFuncName(cmd.Name)
		c.line("builtin.%s(%s, kamiEnv, os.Stdin, os.Stdout, os.Stderr)", funcName, argStr)
	} else {
		c.line("c := exec.Command(%q, %s)", s.Name, argStr)
		c.line("c.Stdin = os.Stdin; c.Stdout = os.Stdout; c.Stderr = os.Stderr")
		c.line("_ = c.Run()")
	}
}

func (c *compiler) compileGoExpression(e *core.GoExpression) string {
	if block, ok := e.Node.(*core.BlockStatement); ok {
		id := fmt.Sprintf("kamiTask_%d", len(c.funcDefs)+1)
		c.addImport("kamishell/recompiler", "")
		c.line("%s := recompiler.NewTask()", id)
		c.line("go func() {")
		c.indent()
		c.line("defer %s.SetResult(nil)", id)
		for _, st := range block.Statements {
			c.compileStatement(st)
		}
		c.dedent()
		c.line("}()")
		return id
	}
	return "nil"
}

func (c *compiler) compileWaitStatement(s *core.WaitStatement) {
	c.addImport("kamishell/recompiler", "")
	if s.Timeout != nil {
		timeout := c.compileExpression(s.Timeout)
		c.line("recompiler.WaitAllTimeout(%s)", timeout)
	} else {
		c.line("recompiler.WaitAll()")
	}
}

func (c *compiler) compilePointerAssignStatement(s *core.PointerAssignStatement) {
	// *p = val — dereference pointer and assign
	// The target is *p (PrefixExpression with Operator="*")
	if prefix, ok := s.Target.(*core.PrefixExpression); ok && prefix.Operator == "*" {
		ptr := c.compileExpression(prefix.Right)
		val := c.compileExpression(s.Value)
		c.addImport("kamishell/recompiler", "")
		c.line("recompiler.SetPtr(%s, %s)", ptr, val)
		return
	}
	c.errorf("invalid pointer assignment target: %T", s.Target)
}

func (c *compiler) compileMethodCallBlockStatement(s *core.MethodCallBlockStatement) {
	obj := c.compileExpression(s.Object)
	switch s.Method {
	case "Go":
		// wg.Go { body } → wg.Go(func() { body })
		// Uses Go 1.26 native sync.WaitGroup.Go
		c.line("%s.Go(func() {", obj)
		c.indent()
		for _, st := range s.Body.Statements {
			c.compileStatement(st)
		}
		c.dedent()
		c.line("})")
	case "Wait":
		// wg.Wait { ... } — not a real pattern, just call Wait
		c.line("%s.Wait()", obj)
	}
}

func (c *compiler) compileExecStatement(s *core.ExecStatement) {
	c.usesEnv = true
	c.addImport("os", "")
	c.addImport("strings", "")
	c.addImport("os/exec", "")
	c.addImport("kamishell/recompiler", "")
	c.addImport("fmt", "")
	cmd := c.compileExpression(s.CommandStr)
	c.line("{")
	c.indent()
	c.line("kamiErr = nil")
	c.line("parts := strings.Fields(recompiler.ToStr(%s))", cmd)
	c.line("if len(parts) > 0 {")
	c.indent()
	c.line("c := exec.Command(parts[0], parts[1:]...)")
	c.line("c.Stdin = os.Stdin; c.Stdout = os.Stdout; c.Stderr = os.Stderr")
	c.line("if err := c.Run(); err != nil { kamiErr = err }")
	c.dedent()
	c.line("}")
	c.line("kamiEnv.SetString(\"err\", recompiler.ToStr(kamiErr))")
	c.dedent()
	c.line("}")
}

func (c *compiler) compileFunctionStatement(s *core.FunctionStatement) {
	funcName := s.Name
	// Track as a known function
	if c.knownFuncs == nil {
		c.knownFuncs = make(map[string]bool)
	}
	c.knownFuncs[funcName] = true

	// Track return type for type inference
	if c.funcReturns == nil {
		c.funcReturns = make(map[string]goType)
	}
	if c.funcReturnList == nil {
		c.funcReturnList = make(map[string][]goType)
	}
	if len(s.ReturnTypes) == 1 {
		c.funcReturns[funcName] = c.kamiTypeToGo(s.ReturnTypes[0])
		c.funcReturnList[funcName] = []goType{c.kamiTypeToGo(s.ReturnTypes[0])}
	} else if len(s.ReturnTypes) > 1 {
		c.funcReturns[funcName] = goAny
		list := make([]goType, len(s.ReturnTypes))
		for i, rt := range s.ReturnTypes {
			list[i] = c.kamiTypeToGo(rt)
		}
		c.funcReturnList[funcName] = list
	} else {
		c.funcReturns[funcName] = goAny
		c.funcReturnList[funcName] = []goType{goAny}
	}

	// Build user parameter list (without kamiEnv — added after body analysis)
	var userParams []string
	for _, p := range s.Parameters {
		goType := c.kamiTypeToGo(p.TypeName)
		userParams = append(userParams, fmt.Sprintf("%s %s", p.Name, goType))
	}

	// Determine return type from declaration
	returnType := "any"
	if len(s.ReturnTypes) == 1 {
		returnType = string(c.kamiTypeToGo(s.ReturnTypes[0]))
	} else if len(s.ReturnTypes) > 1 {
		retTypes := make([]string, len(s.ReturnTypes))
		for i, rt := range s.ReturnTypes {
			retTypes[i] = string(c.kamiTypeToGo(rt))
		}
		returnType = "(" + strings.Join(retTypes, ", ") + ")"
	}

	// Generate function body with a sub-compiler
	sub := &compiler{
		symbols:        make(map[string]goType),
		imports:        c.imports,
		knownFuncs:     c.knownFuncs,
		funcReturns:    c.funcReturns,
		funcReturnList: c.funcReturnList,
		arrayTypes:     copyMap(c.arrayTypes),
		envSync:        c.envSync,
		parentTypes:    c.symbols,
		loopDepth:      0,
		indentLv:       1,
		funcNeedsEnv:   c.funcNeedsEnv,
		waitGroups:     c.waitGroups,
		importedPkgs:   c.importedPkgs,
	}
	// Register parameters with their types so they won't be re-declared
	for _, p := range s.Parameters {
		sub.declareVar(p.Name, c.kamiTypeToGo(p.TypeName))
	}

	// Compile body statements
	if s.Body == nil {
		return
	}
	for _, st := range s.Body.Statements {
		sub.compileStatement(st)
	}

	// Conditionally declare kamiErr (only if 'err' is used in the function)
	kamiErrDecl := ""
	if sub.usesErr {
		kamiErrDecl = "var kamiErr error\n\t_ = kamiErr\n"
	}

	// Check if body ends with a return statement (dead code elimination)
	bodyStr := sub.buf.String()
	hasTrailingReturn := false
	lines := strings.Split(strings.TrimRight(bodyStr, "\n"), "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if strings.HasPrefix(lastLine, "return ") || lastLine == "return" || strings.HasPrefix(lastLine, "return\t") {
			hasTrailingReturn = true
		}
	}

	if !hasTrailingReturn {
		// Add default return only if body doesn't end with return
		if len(s.ReturnTypes) == 0 {
			sub.line("return nil")
		} else if len(s.ReturnTypes) == 1 {
			sub.line("return %s", c.kamiTypeToGo(s.ReturnTypes[0]).zero())
		} else {
			zeros := make([]string, len(s.ReturnTypes))
			for i, rt := range s.ReturnTypes {
				zeros[i] = c.kamiTypeToGo(rt).zero()
			}
			sub.line("return %s", strings.Join(zeros, ", "))
		}
	}

	// Build full parameter list — conditionally add kamiEnv based on body usage
	var params []string
	if c.funcNeedsEnv == nil {
		c.funcNeedsEnv = make(map[string]bool)
	}
	if sub.usesEnv {
		params = append(params, "kamiEnv *recompiler.Env")
		c.funcNeedsEnv[funcName] = true
	} else {
		c.funcNeedsEnv[funcName] = false
	}
	params = append(params, userParams...)
	paramStr := strings.Join(params, ", ")

	// Generate function definition
	var fd strings.Builder
	fd.WriteString(fmt.Sprintf("func kamiFunc_%s(%s) %s {\n", funcName, paramStr, returnType))
	fd.WriteString(kamiErrDecl)
	fd.WriteString(sub.buf.String())
	fd.WriteString("}\n")

	// Register in main env so builtins/commands can find it
	c.usesEnv = true
	c.addImport("kamishell/recompiler", "")
	c.line("kamiEnv.SetString(%q, %q)", funcName, "func")

	// Store func def
	c.funcDefs = append(c.funcDefs, fd.String())
}

func (c *compiler) compileImportStatement(s *core.ImportStatement) {
	path := s.Path
	// Map "Go/fmt" → "fmt", "Go/strings" → "strings", etc.
	if strings.HasPrefix(path, "Go/") {
		goPkg := strings.TrimPrefix(path, "Go/")
		c.addImport(goPkg, "")
		if c.importedPkgs == nil {
			c.importedPkgs = make(map[string]bool)
		}
		c.importedPkgs[goPkg] = true
	} else if path == "Go" {
		// "Go" alone doesn't need an import
	}
}

// Helper functions

func (c *compiler) kamiTypeToGo(kamiType string) goType {
	switch strings.ToLower(kamiType) {
	case "int", "integer":
		return goInt
	case "float", "float64":
		return goFloat
	case "string":
		return goStr
	case "bool", "boolean":
		return goBool
	case "array":
		return goArrAny
	}
	return goAny
}

func (c *compiler) inferGoType(expr core.Expression) goType {
	switch e := expr.(type) {
	case *core.IntegerLiteral:
		return goInt
	case *core.FloatLiteral:
		return goFloat
	case *core.StringLiteral:
		return goStr
	case *core.BooleanLiteral:
		return goBool
	case *core.NilLiteral:
		return goAny
	case *core.ArrayLiteral:
		if len(e.Elements) == 0 {
			return goArrAny
		}
		// Infer typed array if all elements have the same type
		elemType := c.inferGoType(e.Elements[0])
		allSame := true
		for _, el := range e.Elements[1:] {
			if c.inferGoType(el) != elemType {
				allSame = false
				break
			}
		}
		if allSame {
			switch elemType {
			case goInt:
				return "[]int64"
			case goStr:
				return "[]string"
			case goFloat:
				return "[]float64"
			case goBool:
				return "[]bool"
			}
		}
		return goArrAny
	case *core.InfixExpression:
		if e.Operator == "+" || e.Operator == "-" || e.Operator == "*" || e.Operator == "/" {
			if c.isType(e.Left, goInt) && c.isType(e.Right, goInt) {
				return goInt
			}
		}
		if e.Operator == "==" || e.Operator == "!=" || e.Operator == "<" || e.Operator == ">" || e.Operator == "<=" || e.Operator == ">=" {
			return goBool
		}
		return goAny
	case *core.PrefixExpression:
		if e.Operator == "!" {
			return goBool
		}
	case *core.CallExpression:
		// Look up return type of known functions
		if id, ok := e.Function.(*core.Identifier); ok && c.funcReturns != nil {
			if retType, exists := c.funcReturns[id.Value]; exists {
				return retType
			}
		}
		return goAny
	case *core.IndexExpression:
		// Infer element type from array
		if elemType, ok := c.inferArrayType(e.Left); ok {
			return elemType
		}
		return goAny
	case *core.Identifier:
		if e.Value == "nil" || e.Value == "true" || e.Value == "false" {
			if e.Value == "true" || e.Value == "false" {
				return goBool
			}
			return goAny
		}
		if t, ok := c.getVarType(e.Value); ok {
			return t
		}
	case *core.FunctionLiteral:
		return goAny
	}
	return goAny
}

// inferArrayType returns the element type of an array expression if known.
func (c *compiler) inferArrayType(expr core.Expression) (goType, bool) {
	switch e := expr.(type) {
	case *core.ArrayLiteral:
		if len(e.Elements) == 0 {
			return goAny, false
		}
		elemType := c.inferGoType(e.Elements[0])
		// Check all elements have the same type
		for _, el := range e.Elements[1:] {
			if c.inferGoType(el) != elemType {
				return goAny, false
			}
		}
		return elemType, true
	case *core.Identifier:
		// Check variable type directly
		if t, ok := c.getVarType(e.Value); ok {
			switch t {
			case "[]int64":
				return goInt, true
			case "[]string":
				return goStr, true
			case "[]float64":
				return goFloat, true
			case "[]bool":
				return goBool, true
			case goArrAny:
				// Check arrayTypes map for more specific type
				if c.arrayTypes != nil {
					if elemType, exists := c.arrayTypes[e.Value]; exists {
						return elemType, true
					}
				}
			}
		}
	}
	return goAny, false
}

func (c *compiler) isType(expr core.Expression, t goType) bool {
	switch e := expr.(type) {
	case *core.IntegerLiteral:
		return t == goInt
	case *core.FloatLiteral:
		return t == goFloat
	case *core.StringLiteral:
		return t == goStr
	case *core.BooleanLiteral:
		return t == goBool
	case *core.NilLiteral:
		return false
	case *core.Identifier:
		if vt, ok := c.getVarType(e.Value); ok {
			return vt == t
		}
		return false
	case *core.CallExpression:
		return c.inferGoType(expr) == t
	case *core.InfixExpression:
		return c.inferGoType(expr) == t
	case *core.IndexExpression:
		return c.inferGoType(expr) == t
	}
	return false
}

func (c *compiler) builtinFuncName(name string) string {
	switch name {
	case "cd":
		return "Cd"
	case "pwd":
		return "Pwd"
	case "ls":
		return "Ls"
	case "cat":
		return "Cat"
	case "rm":
		return "Rm"
	case "mv":
		return "Mv"
	case "cp":
		return "Cp"
	case "mkdir":
		return "Mkdir"
	case "touch":
		return "Touch"
	case "grep":
		return "Grep"
	case "sed":
		return "Sed"
	case "which":
		return "Which"
	case "type":
		return "Type"
	case "help":
		return "Help"
	case "export":
		return "Export"
	case "env":
		return "Env"
	case "exit":
		return "Exit"
	case "jobs":
		return "JobsCmd"
	case "http":
		return "HTTP"
	case "make":
		return "Make"
	default:
		return "Unknown"
	}
}

func copyMap(src map[string]goType) map[string]goType {
	if src == nil {
		return make(map[string]goType)
	}
	dst := make(map[string]goType, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}