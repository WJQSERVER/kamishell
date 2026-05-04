package recompiler

import (
	"fmt"
	"go/format"
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
	buf        strings.Builder
	imports    map[string]string
	symbols    map[string]goType
	funcDefs   []string
	knownFuncs map[string]bool
	loopDepth  int
	indentLv   int
	err        error
	hasErr     bool
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

func Compile(program *core.Program) (*CompiledScript, error) {
	comp := &compiler{
		symbols: make(map[string]goType),
	}

	// Track whether kamiErr/kamiEnv are actually used
	comp.buf.WriteString("func kami_main() {\n")
	comp.indentLv = 1

	comp.line("var kamiErr error")
	comp.line("kamiEnv := recompiler.NewEnv()")
	comp.addImport("kamishell/recompiler", "")
	// Suppress unused errors if no commands run
	comp.line("_ = kamiErr")
	comp.line("_ = kamiEnv")

	for _, stmt := range program.Statements {
		comp.compileStatement(stmt)
		if comp.hasErr {
			return nil, comp.err
		}
	}

	comp.line("")
	comp.dedent()
	comp.buf.WriteString("}\n")

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
		c.line("_ = %s", val)
	}
}

func (c *compiler) compilePrintStatement(s *core.PrintStatement) {
	c.addImport("fmt", "")
	val := c.compileExpression(s.Expression)

	// Direct output for simple string literals (no interpolation)
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		raw, _ := strconv.Unquote(val)
		if !strings.Contains(raw, "$") {
			c.line("fmt.Println(%s)", val)
			return
		}
	}

	// Direct strconv for known types — skip recompiler.ToStr(any) overhead
	expr := s.Expression
	switch {
	case c.isType(expr, goStr):
		c.line("fmt.Println(%s)", val)
	case c.isType(expr, goInt):
		c.addImport("strconv", "")
		c.line("fmt.Println(strconv.FormatInt(%s, 10))", val)
	case c.isType(expr, goBool):
		c.addImport("strconv", "")
		c.line("fmt.Println(strconv.FormatBool(%s))", val)
	case c.isType(expr, goFloat):
		c.addImport("strconv", "")
		c.line("fmt.Println(strconv.FormatFloat(%s, 'f', -1, 64))", val)
	default:
		c.addImport("kamishell/recompiler", "")
		c.line("fmt.Println(recompiler.ToStr(%s))", val)
	}
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
				args = append(args, "kamiEnv")
				for _, a := range ce.Arguments {
					args = append(args, c.compileExpression(a))
				}
				callStr := fmt.Sprintf("kamiFunc_%s(%s)", id.Value, strings.Join(args, ", "))
				c.line("_ = %s // multi-return unpacking TODO", callStr)
				for _, name := range s.Names {
					c.declareVar(name, goAny)
					c.line("var %s any", name)
				}
				return
			}
		}
		// Fallback: single value unpacked into multiple names
		for i, name := range s.Names {
			c.declareVar(name, goAny)
			c.line("var %s any", name)
			c.line("_ = %s // unpack[%d] TODO", val, i)
		}
		return
	}

	if _, ok := s.Value.(*core.FunctionLiteral); ok {
		c.declareVar(s.Names[0], goAny)
		c.line("var %s any = %s", s.Names[0], val)
		c.line("kamiEnv.Set(%q, %s)", s.Names[0], s.Names[0])
		return
	}

	if s.Token.Literal == ":=" {
		// Variable declaration: infer type from expr
		typ := c.inferGoType(s.Value)
		c.declareVar(s.Names[0], typ)
		c.line("var %s %s = %s", s.Names[0], typ, val)
		c.line("kamiEnv.Set(%q, %s)", s.Names[0], s.Names[0])
	} else {
		// Reassignment
		if !c.hasVar(s.Names[0]) {
			c.declareVar(s.Names[0], goAny)
		}
		c.line("%s = %s", s.Names[0], val)
		c.line("kamiEnv.Set(%q, %s)", s.Names[0], s.Names[0])
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
	c.line("kamiEnv.Set(%q, %s)", s.Name, s.Name)
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
		if c.hasVar(a.Names[0]) {
			buf.WriteString(fmt.Sprintf("; kamiEnv.Set(%q, %s)", a.Names[0], a.Names[0]))
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
	c.addImport("kamishell/recompiler", "")
	if s.Tag != nil {
		tag := c.compileExpression(s.Tag)
		c.line("switch recompiler.ToStr(%s) {", tag)
	} else {
		c.line("switch {")
	}
	c.indent()
	for _, cas := range s.Cases {
		if cas.Values == nil {
			// default
			c.line("default:")
		} else {
			var vals []string
			for _, v := range cas.Values {
				vals = append(vals, c.compileExpression(v))
			}
			// Use string comparison in generated code
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
		c.addImport("kamishell/recompiler", "")
		return fmt.Sprintf("recompiler.ExpandStr(%q, kamiEnv)", s.Value)
	}
	// strconv.Quote is used at recompiler-compile-time to produce valid Go source.
	// The generated code contains the quoted string directly, not a strconv.Quote call.
	return strconv.Quote(s.Value)
}

func (c *compiler) compileIdentifier(id *core.Identifier) string {
	name := id.Value
	if name == "err" {
		return "kamiErr"
	}
	if name == "nil" {
		return "nil"
	}
	if name == "true" {
		return "true"
	}
	if name == "false" {
		return "false"
	}
	if c.hasVar(name) {
		return name
	}
	// Known user-defined functions: reference the Go function directly
	if c.knownFuncs != nil && c.knownFuncs[name] {
		return fmt.Sprintf("kamiFunc_%s", name)
	}
	c.addImport("kamishell/recompiler", "")
	c.declareVar(name, goAny)
	c.line("var %s any", name)
	c.line("if v, ok := kamiEnv.Get(%q); ok { %s = v }", name, name)
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
	c.addImport("kamishell/recompiler", "")
	right := c.compileExpression(e.Right)
	switch e.Operator {
	case "!":
		return fmt.Sprintf("!recompiler.IsTruthy(%s)", right)
	case "-":
		return fmt.Sprintf("(-%s)", right)
	case "&":
		return fmt.Sprintf("recompiler.ToStr(%s)", right)
	case "*":
		return right
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
		}

		// Known user-defined functions: call directly
		if c.knownFuncs != nil && c.knownFuncs[name] {
			c.addImport("kamishell/recompiler", "")
			var args []string
			args = append(args, "kamiEnv")
			for _, a := range e.Arguments {
				args = append(args, c.compileExpression(a))
			}
			return fmt.Sprintf("kamiFunc_%s(%s)", name, strings.Join(args, ", "))
		}
	}

	// Generic call: function loaded from env as any
	c.addImport("kamishell/recompiler", "")
	fn := c.compileExpression(e.Function)
	var args []string
	for _, a := range e.Arguments {
		args = append(args, c.compileExpression(a))
	}
	return fmt.Sprintf("recompiler.CallFunc(%s, kamiEnv, %s)", fn, strings.Join(args, ", "))
}

func (c *compiler) compileMemberExpression(e *core.MemberExpression) string {
	c.addImport("kamishell/recompiler", "")
	obj := c.compileExpression(e.Object)
	prop := e.Property

	if id, ok := e.Object.(*core.Identifier); ok {
		switch id.Value {
		case "env":
			return fmt.Sprintf("kamiEnv.GetStr(%q)", prop)
		case "param":
			return fmt.Sprintf("recompiler.ToStr(paramGet(%q))", prop)
		}
	}

	return fmt.Sprintf("recompiler.MemberGet(%s, %q)", obj, prop)
}

func (c *compiler) compileArrayLiteral(a *core.ArrayLiteral) string {
	if len(a.Elements) == 0 {
		return "[]any{}"
	}
	var elems []string
	for _, el := range a.Elements {
		elems = append(elems, c.compileExpression(el))
	}
	return fmt.Sprintf("[]any{%s}", strings.Join(elems, ", "))
}

func (c *compiler) compileFunctionLiteral(f *core.FunctionLiteral) string {
	var params []string
	for _, p := range f.Parameters {
		goType := c.kamiTypeToGo(p.TypeName)
		params = append(params, fmt.Sprintf("%s %s", p.Name, goType))
	}
	paramStr := strings.Join(params, ", ")

	// Generate unique function name for closures
	id := fmt.Sprintf("closure_%d", len(c.funcDefs)+1)

	var bodyBuf strings.Builder
	sub := &compiler{
		symbols:   make(map[string]goType),
		imports:   c.imports,
		loopDepth: c.loopDepth,
	}
	sub.line("func(%s any) any {", paramStr)
	sub.indent()
	for _, st := range f.Body.Statements {
		sub.compileStatement(st)
	}
	sub.line("return nil")
	sub.dedent()
	sub.line("}")

	return bodyBuf.String()
	_ = id
	return fmt.Sprintf("func(%s any) any { return nil }", paramStr)
}

func (c *compiler) compileCommandStatement(s *core.CommandStatement) {
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
		c.line("kamiEnv.Set(\"err\", kamiErr)")
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
	c.line("kamiEnv.Set(\"err\", kamiErr)")
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
	c.line("kamiEnv.Set(\"err\", kamiErr)")
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

	c.line("kamiEnv.Set(\"err\", kamiErr)")
	c.dedent()
	c.line("}")
}

func (c *compiler) compilePipeWithOutput(ps *core.PipeStatement, output string) {
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

func (c *compiler) compileExecStatement(s *core.ExecStatement) {
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
	c.line("kamiEnv.Set(\"err\", kamiErr)")
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

	var params []string
	params = append(params, "kamiEnv *recompiler.Env")
	for _, p := range s.Parameters {
		goType := c.kamiTypeToGo(p.TypeName)
		params = append(params, fmt.Sprintf("%s %s", p.Name, goType))
	}
	paramStr := strings.Join(params, ", ")

	// Determine return type
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
		symbols:    make(map[string]goType),
		imports:    c.imports,
		knownFuncs: c.knownFuncs,
		loopDepth:  0,
		indentLv:   1,
	}
	// Register parameters with their types so they won't be re-declared
	for _, p := range s.Parameters {
		sub.declareVar(p.Name, c.kamiTypeToGo(p.TypeName))
	}

	// Compile body statements
	for _, st := range s.Body.Statements {
		sub.compileStatement(st)
	}
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

	// Generate function definition
	var fd strings.Builder
	fd.WriteString(fmt.Sprintf("func kamiFunc_%s(%s) %s {\n", funcName, paramStr, returnType))
	fd.WriteString(sub.buf.String())
	fd.WriteString("}\n")

	// Register in main env so builtins/commands can find it
	c.addImport("kamishell/recompiler", "")
	c.line("kamiEnv.Set(%q, any(kamiFunc_%s))", funcName, funcName)

	// Store func def
	c.funcDefs = append(c.funcDefs, fd.String())
}

func (c *compiler) compileImportStatement(s *core.ImportStatement) {
	path := s.Path
	// Map "Go/fmt" → "fmt", "Go/strings" → "strings", etc.
	if strings.HasPrefix(path, "Go/") {
		goPkg := strings.TrimPrefix(path, "Go/")
		c.addImport(goPkg, "")
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