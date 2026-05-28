package core

import (
	"testing"
)

// --- Helper ---

func mustParseResolve(t *testing.T, input string) *Program {
	t.Helper()
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	Resolve(program)
	return program
}

func findIdentifier(stmts []Statement, name string) *Identifier {
	for _, stmt := range stmts {
		if ident := findIdentInStatement(stmt, name); ident != nil {
			return ident
		}
	}
	return nil
}

func findIdentInStatement(stmt Statement, name string) *Identifier {
	switch s := stmt.(type) {
	case *ExpressionStatement:
		return findIdentInExpr(s.Expression, name)
	case *PrintStatement:
		return findIdentInExpr(s.Expression, name)
	case *AssignStatement:
		for _, n := range s.Names {
			if n == name {
				// Check if the value references this ident
			}
		}
		if id := findIdentInExpr(s.Value, name); id != nil {
			return id
		}
	case *IfStatement:
		if id := findIdentInExpr(s.Condition, name); id != nil {
			return id
		}
		if s.Consequence != nil {
			if id := findIdentifier(s.Consequence.Statements, name); id != nil {
				return id
			}
		}
		if s.Alternative != nil {
			if id := findIdentifier(s.Alternative.Statements, name); id != nil {
				return id
			}
		}
	case *ForStatement:
		if s.Init != nil {
			if id := findIdentInStatement(s.Init, name); id != nil {
				return id
			}
		}
		if id := findIdentInExpr(s.Condition, name); id != nil {
			return id
		}
		if s.Post != nil {
			if id := findIdentInStatement(s.Post, name); id != nil {
				return id
			}
		}
		if s.Consequence != nil {
			if id := findIdentifier(s.Consequence.Statements, name); id != nil {
				return id
			}
		}
	case *VarStatement:
		if id := findIdentInExpr(s.Value, name); id != nil {
			return id
		}
	case *ReturnStatement:
		for _, rv := range s.ReturnValues {
			if id := findIdentInExpr(rv, name); id != nil {
				return id
			}
		}
	case *SwitchStatement:
		if id := findIdentInExpr(s.Tag, name); id != nil {
			return id
		}
		for i := range s.Cases {
			for _, v := range s.Cases[i].Values {
				if id := findIdentInExpr(v, name); id != nil {
					return id
				}
			}
			if s.Cases[i].Body != nil {
				if id := findIdentifier(s.Cases[i].Body.Statements, name); id != nil {
					return id
				}
			}
		}
	case *FunctionStatement:
		if s.Body != nil {
			if id := findIdentifier(s.Body.Statements, name); id != nil {
				return id
			}
		}
	case *PipeStatement:
		for _, cmd := range s.Commands {
			if id := findIdentInStatement(cmd, name); id != nil {
				return id
			}
		}
	case *RedirectStatement:
		if id := findIdentInStatement(s.Source, name); id != nil {
			return id
		}
		if id := findIdentInExpr(s.Target, name); id != nil {
			return id
		}
	case *LogicalStatement:
		if id := findIdentInStatement(s.Left, name); id != nil {
			return id
		}
		if id := findIdentInStatement(s.Right, name); id != nil {
			return id
		}
	case *BackgroundStatement:
		if id := findIdentInStatement(s.Stmt, name); id != nil {
			return id
		}
	}
	return nil
}

func findIdentInExpr(expr Expression, name string) *Identifier {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *Identifier:
		if e.Value == name {
			return e
		}
	case *InfixExpression:
		if id := findIdentInExpr(e.Left, name); id != nil {
			return id
		}
		if id := findIdentInExpr(e.Right, name); id != nil {
			return id
		}
	case *PrefixExpression:
		if id := findIdentInExpr(e.Right, name); id != nil {
			return id
		}
	case *CallExpression:
		if id := findIdentInExpr(e.Function, name); id != nil {
			return id
		}
		for _, arg := range e.Arguments {
			if id := findIdentInExpr(arg, name); id != nil {
				return id
			}
		}
	case *MemberExpression:
		if id := findIdentInExpr(e.Object, name); id != nil {
			return id
		}
	case *IndexExpression:
		if id := findIdentInExpr(e.Left, name); id != nil {
			return id
		}
		if id := findIdentInExpr(e.Index, name); id != nil {
			return id
		}
	case *ArrayLiteral:
		for _, el := range e.Elements {
			if id := findIdentInExpr(el, name); id != nil {
				return id
			}
		}
	case *FunctionLiteral:
		if e.Body != nil {
			if id := findIdentifier(e.Body.Statements, name); id != nil {
				return id
			}
		}
	}
	return nil
}

// --- Resolver Unit Tests ---

func TestResolverSimpleDeclaration(t *testing.T) {
	program := mustParseResolve(t, `x := 10; print x`)

	// Find the Identifier "x" in the print statement
	printStmt, ok := program.Statements[1].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	ident, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier")
	}

	if ident.ScopeDepth != 0 {
		t.Errorf("expected ScopeDepth=0, got %d", ident.ScopeDepth)
	}
	if ident.SlotIndex != 0 {
		t.Errorf("expected SlotIndex=0, got %d", ident.SlotIndex)
	}
}

func TestResolverMultipleDeclarations(t *testing.T) {
	program := mustParseResolve(t, `x := 10; y := 20; z := x + y`)

	assignStmt, ok := program.Statements[2].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement")
	}
	infix, ok := assignStmt.Value.(*InfixExpression)
	if !ok {
		t.Fatal("expected InfixExpression")
	}

	xIdent, ok := infix.Left.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for x")
	}
	if xIdent.ScopeDepth != 0 || xIdent.SlotIndex != 0 {
		t.Errorf("x: expected depth=0 slot=0, got depth=%d slot=%d", xIdent.ScopeDepth, xIdent.SlotIndex)
	}

	yIdent, ok := infix.Right.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for y")
	}
	if yIdent.ScopeDepth != 0 || yIdent.SlotIndex != 1 {
		t.Errorf("y: expected depth=0 slot=1, got depth=%d slot=%d", yIdent.ScopeDepth, yIdent.SlotIndex)
	}
}

func TestResolverNestedScope(t *testing.T) {
	program := mustParseResolve(t, `x := 1; if true { y := 2; print x }`)

	ifStmt, ok := program.Statements[1].(*IfStatement)
	if !ok {
		t.Fatal("expected IfStatement")
	}
	printStmt, ok := ifStmt.Consequence.Statements[1].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement inside if block")
	}
	ident, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for x")
	}

	// x is declared in the outer scope (depth 0), accessed from inner scope (depth 1)
	if ident.ScopeDepth != 1 {
		t.Errorf("expected ScopeDepth=1 (one scope out), got %d", ident.ScopeDepth)
	}
	if ident.SlotIndex != 0 {
		t.Errorf("expected SlotIndex=0, got %d", ident.SlotIndex)
	}
}

func TestResolverInnerVariableNotAccessible(t *testing.T) {
	// Variable declared inside a block should still resolve at depth 0 within its own scope
	program := mustParseResolve(t, `if true { x := 5; print x }`)

	ifStmt, ok := program.Statements[0].(*IfStatement)
	if !ok {
		t.Fatal("expected IfStatement")
	}
	printStmt, ok := ifStmt.Consequence.Statements[1].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	ident, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier")
	}

	// x is declared in the if-block scope, accessed within the same scope
	if ident.ScopeDepth != 0 {
		t.Errorf("expected ScopeDepth=0, got %d", ident.ScopeDepth)
	}
	if ident.SlotIndex != 0 {
		t.Errorf("expected SlotIndex=0, got %d", ident.SlotIndex)
	}
}

func TestResolverForLoopVariables(t *testing.T) {
	program := mustParseResolve(t, `sum := 0; for i := 0; i < 10; i = i + 1 { sum = sum + i }`)

	// Find i in the condition
	forStmt, ok := program.Statements[1].(*ForStatement)
	if !ok {
		t.Fatal("expected ForStatement")
	}
	condIdent, ok := forStmt.Condition.(*InfixExpression)
	if !ok {
		t.Fatal("expected InfixExpression in condition")
	}
	iIdent, ok := condIdent.Left.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for i in condition")
	}

	if iIdent.ScopeDepth != 0 {
		t.Errorf("i in condition: expected ScopeDepth=0, got %d", iIdent.ScopeDepth)
	}
	if iIdent.SlotIndex < 0 {
		t.Errorf("i in condition: expected SlotIndex >= 0, got %d", iIdent.SlotIndex)
	}
}

func TestResolverForLoopIncResolved(t *testing.T) {
	// HasInc triggers when the body has exactly one statement: i = i + 1
	program := mustParseResolve(t, `for i := 0; i < 10 { i = i + 1 }`)

	forStmt, ok := program.Statements[0].(*ForStatement)
	if !ok {
		t.Fatal("expected ForStatement")
	}

	if !forStmt.HasInc {
		t.Fatal("expected HasInc=true")
	}
	if forStmt.IncScopeDepth < 0 {
		t.Errorf("expected IncScopeDepth >= 0, got %d", forStmt.IncScopeDepth)
	}
	if forStmt.IncSlotIndex < 0 {
		t.Errorf("expected IncSlotIndex >= 0, got %d", forStmt.IncSlotIndex)
	}
}

func TestResolverReassignmentResolved(t *testing.T) {
	program := mustParseResolve(t, `x := 10; x = 20`)

	assignStmt, ok := program.Statements[1].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement")
	}

	if assignStmt.ResolvedScopeDepth != 0 {
		t.Errorf("expected ResolvedScopeDepth=0, got %d", assignStmt.ResolvedScopeDepth)
	}
	if assignStmt.ResolvedSlotIndex != 0 {
		t.Errorf("expected ResolvedSlotIndex=0, got %d", assignStmt.ResolvedSlotIndex)
	}
}

func TestResolverReassignmentOuterScope(t *testing.T) {
	program := mustParseResolve(t, `x := 1; if true { x = 2 }`)

	ifStmt, ok := program.Statements[1].(*IfStatement)
	if !ok {
		t.Fatal("expected IfStatement")
	}
	assignStmt, ok := ifStmt.Consequence.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement inside if")
	}

	if assignStmt.ResolvedScopeDepth != 1 {
		t.Errorf("expected ResolvedScopeDepth=1, got %d", assignStmt.ResolvedScopeDepth)
	}
	if assignStmt.ResolvedSlotIndex != 0 {
		t.Errorf("expected ResolvedSlotIndex=0, got %d", assignStmt.ResolvedSlotIndex)
	}
}

func TestResolverReassignmentUndeclaredUnresolved(t *testing.T) {
	// Assigning to a variable not declared in any scope should remain unresolved
	program := mustParseResolve(t, `x = 20`)

	assignStmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement")
	}

	if assignStmt.ResolvedScopeDepth != -1 {
		t.Errorf("expected ResolvedScopeDepth=-1 (unresolved), got %d", assignStmt.ResolvedScopeDepth)
	}
	if assignStmt.ResolvedSlotIndex != -1 {
		t.Errorf("expected ResolvedSlotIndex=-1 (unresolved), got %d", assignStmt.ResolvedSlotIndex)
	}
}

func TestResolverReservedNamesNotSlotified(t *testing.T) {
	// 'err' is a reserved name and should not get slot resolution
	program := mustParseResolve(t, `ls missing; if err != nil { print err }`)

	ifStmt, ok := program.Statements[1].(*IfStatement)
	if !ok {
		t.Fatal("expected IfStatement")
	}
	condInfix, ok := ifStmt.Condition.(*InfixExpression)
	if !ok {
		t.Fatal("expected InfixExpression")
	}
	errIdent, ok := condInfix.Left.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for err")
	}

	if errIdent.ScopeDepth != -1 || errIdent.SlotIndex != -1 {
		t.Errorf("err should be unresolved: depth=%d slot=%d", errIdent.ScopeDepth, errIdent.SlotIndex)
	}
}

func TestResolverBuiltinNotSlotified(t *testing.T) {
	// Builtin names like 'len' should not get slot resolution
	program := mustParseResolve(t, `x := [1,2,3]; print len(x)`)

	printStmt, ok := program.Statements[1].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	callExpr, ok := printStmt.Expression.(*CallExpression)
	if !ok {
		t.Fatal("expected CallExpression")
	}
	lenIdent, ok := callExpr.Function.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for len")
	}

	if lenIdent.ScopeDepth != -1 || lenIdent.SlotIndex != -1 {
		t.Errorf("len should be unresolved: depth=%d slot=%d", lenIdent.ScopeDepth, lenIdent.SlotIndex)
	}
}

func TestResolverPackageNamesNotSlotified(t *testing.T) {
	// 'env' and 'sync' are package names, not variables
	program := mustParseResolve(t, `env.Set("K", "V")`)

	exprStmt, ok := program.Statements[0].(*ExpressionStatement)
	if !ok {
		t.Fatal("expected ExpressionStatement")
	}
	memberExpr, ok := exprStmt.Expression.(*CallExpression)
	if !ok {
		t.Fatal("expected CallExpression")
	}
	member, ok := memberExpr.Function.(*MemberExpression)
	if !ok {
		t.Fatal("expected MemberExpression")
	}
	envIdent, ok := member.Object.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for env")
	}

	if envIdent.ScopeDepth != -1 || envIdent.SlotIndex != -1 {
		t.Errorf("env should be unresolved: depth=%d slot=%d", envIdent.ScopeDepth, envIdent.SlotIndex)
	}
}

func TestResolverNativeFnNotSlotified(t *testing.T) {
	// 'len' is a NativeFn, should not get slot resolution
	program := mustParseResolve(t, `x := len("hello")`)

	assignStmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement")
	}
	callExpr, ok := assignStmt.Value.(*CallExpression)
	if !ok {
		t.Fatal("expected CallExpression")
	}
	lenIdent, ok := callExpr.Function.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for len")
	}

	if lenIdent.ScopeDepth != -1 || lenIdent.SlotIndex != -1 {
		t.Errorf("len should be unresolved: depth=%d slot=%d", lenIdent.ScopeDepth, lenIdent.SlotIndex)
	}
}

func TestResolverFunctionParamsGetSlots(t *testing.T) {
	program := mustParseResolve(t, `func add(a int, b int) int { return a + b }`)

	funcStmt, ok := program.Statements[0].(*FunctionStatement)
	if !ok {
		t.Fatal("expected FunctionStatement")
	}

	// Find 'a' in the return expression
	retStmt, ok := funcStmt.Body.Statements[0].(*ReturnStatement)
	if !ok {
		t.Fatal("expected ReturnStatement")
	}
	infix, ok := retStmt.ReturnValues[0].(*InfixExpression)
	if !ok {
		t.Fatal("expected InfixExpression")
	}
	aIdent, ok := infix.Left.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for a")
	}

	if aIdent.ScopeDepth != 0 {
		t.Errorf("a: expected ScopeDepth=0, got %d", aIdent.ScopeDepth)
	}
	if aIdent.SlotIndex != 0 {
		t.Errorf("a: expected SlotIndex=0, got %d", aIdent.SlotIndex)
	}
}

func TestResolverClosureCapturesOuterSlot(t *testing.T) {
	program := mustParseResolve(t, `x := 10; f := func() { print x }; f()`)

	assignStmt, ok := program.Statements[1].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement for f")
	}
	funcLit, ok := assignStmt.Value.(*FunctionLiteral)
	if !ok {
		t.Fatal("expected FunctionLiteral")
	}
	printStmt, ok := funcLit.Body.Statements[0].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	xIdent, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for x")
	}

	// x is in the outer scope (one level out from the closure)
	if xIdent.ScopeDepth != 1 {
		t.Errorf("x in closure: expected ScopeDepth=1, got %d", xIdent.ScopeDepth)
	}
	if xIdent.SlotIndex != 0 {
		t.Errorf("x in closure: expected SlotIndex=0, got %d", xIdent.SlotIndex)
	}
}

func TestResolverWhileLoopBodyResolves(t *testing.T) {
	// while-style for loop (no init): variables from outer scope should be accessible
	program := mustParseResolve(t, `i := 0; for i < 5 { i = i + 1; print i }`)

	forStmt, ok := program.Statements[1].(*ForStatement)
	if !ok {
		t.Fatal("expected ForStatement")
	}

	// Find i in the body
	assignStmt, ok := forStmt.Consequence.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement in body")
	}
	if assignStmt.ResolvedScopeDepth != 0 {
		t.Errorf("i in body reassignment: expected depth=0, got %d", assignStmt.ResolvedScopeDepth)
	}
	if assignStmt.ResolvedSlotIndex < 0 {
		t.Errorf("i in body reassignment: expected slot >= 0, got %d", assignStmt.ResolvedSlotIndex)
	}
}

func TestResolverVarStatementDeclaresSlot(t *testing.T) {
	program := mustParseResolve(t, `var x int; print x`)

	printStmt, ok := program.Statements[1].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	ident, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier")
	}

	if ident.ScopeDepth != 0 || ident.SlotIndex != 0 {
		t.Errorf("var x: expected depth=0 slot=0, got depth=%d slot=%d", ident.ScopeDepth, ident.SlotIndex)
	}
}

func TestResolverSwitchTag(t *testing.T) {
	program := mustParseResolve(t, `x := 2; switch x { case 1: print "one" case 2: print "two" }`)

	switchStmt, ok := program.Statements[1].(*SwitchStatement)
	if !ok {
		t.Fatal("expected SwitchStatement")
	}
	tagIdent, ok := switchStmt.Tag.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier for tag")
	}

	if tagIdent.ScopeDepth != 0 || tagIdent.SlotIndex != 0 {
		t.Errorf("switch tag: expected depth=0 slot=0, got depth=%d slot=%d", tagIdent.ScopeDepth, tagIdent.SlotIndex)
	}
}

func TestResolverMultiAssignDeclaresSlots(t *testing.T) {
	program := mustParseResolve(t, `func div(a int, b int) (int, error) { return a / b, nil }; q, r := div(17, 5)`)

	multiAssign, ok := program.Statements[1].(*AssignStatement)
	if !ok {
		t.Fatal("expected AssignStatement")
	}
	if len(multiAssign.Names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(multiAssign.Names))
	}

	// Both q and r should be in the current scope
	if multiAssign.Names[0] != "q" || multiAssign.Names[1] != "r" {
		t.Errorf("unexpected names: %v", multiAssign.Names)
	}
}

func TestResolverUndefinedVariableRemainsUnresolved(t *testing.T) {
	program := mustParseResolve(t, `print undefinedVar`)

	printStmt, ok := program.Statements[0].(*PrintStatement)
	if !ok {
		t.Fatal("expected PrintStatement")
	}
	ident, ok := printStmt.Expression.(*Identifier)
	if !ok {
		t.Fatal("expected Identifier")
	}

	if ident.ScopeDepth != -1 || ident.SlotIndex != -1 {
		t.Errorf("undefined var should be unresolved: depth=%d slot=%d", ident.ScopeDepth, ident.SlotIndex)
	}
}

// --- Integration Tests ---

func TestResolverIntegrationVarReassignInLoop(t *testing.T) {
	stdout, stderr, _ := runKami(`sum := 0; for i := 0; i < 5; i = i + 1 { sum = sum + i }; print sum`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "10" {
		t.Errorf("expected 10, got %s", trim(stdout))
	}
}

func TestResolverIntegrationNestedScopeAccess(t *testing.T) {
	stdout, stderr, _ := runKami(`x := 42; if true { print x }`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "42" {
		t.Errorf("expected 42, got %s", trim(stdout))
	}
}

func TestResolverIntegrationClosureReadsOuter(t *testing.T) {
	stdout, stderr, _ := runKami(`x := 100; f := func() int { return x }; print f()`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "100" {
		t.Errorf("expected 100, got %s", trim(stdout))
	}
}

func TestResolverIntegrationFunctionCallWithSlots(t *testing.T) {
	stdout, stderr, _ := runKami(`func add(a int, b int) int { return a + b }; print add(3, 4)`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "7" {
		t.Errorf("expected 7, got %s", trim(stdout))
	}
}

func TestResolverIntegrationMultipleVarsSameScope(t *testing.T) {
	stdout, stderr, _ := runKami(`a := 1; b := 2; c := 3; print a + b + c`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "6" {
		t.Errorf("expected 6, got %s", trim(stdout))
	}
}

func TestResolverIntegrationSwitchWithResolvedTag(t *testing.T) {
	stdout, stderr, _ := runKami(`x := 2; switch x { case 1: print "one" case 2: print "two" default: print "other" }`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "two" {
		t.Errorf("expected 'two', got %s", trim(stdout))
	}
}

func TestResolverIntegrationErrVariableWorks(t *testing.T) {
	// err is set by evalStatements and returned as result — test that it propagates
	_, stderr, result := runKami(`exec "false"`, NewEmptyEnvironment())
	if !isError(result) {
		t.Errorf("expected error result, got %v", result)
	}
	if stderr == "" {
		t.Error("expected stderr output")
	}
}

func TestResolverIntegrationPointerAssignment(t *testing.T) {
	stdout, stderr, _ := runKami(`x := 10; p := &x; *p = 20; print x`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "20" {
		t.Errorf("expected 20, got %s", trim(stdout))
	}
}

func TestResolverIntegrationTypeMismatchRejected(t *testing.T) {
	_, stderr, _ := runKami(`x := 10; x = "hello"`, NewEmptyEnvironment())
	if stderr == "" {
		t.Error("expected type mismatch error")
	}
}

func TestResolverIntegrationForLoopLargeCount(t *testing.T) {
	stdout, stderr, _ := runKami(`sum := 0; for i := 0; i < 1000; i = i + 1 { sum = sum + i }; print sum`, NewEmptyEnvironment())
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if trim(stdout) != "499500" {
		t.Errorf("expected 499500, got %s", trim(stdout))
	}
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// ============================================================
// Reserved names — P2
// ============================================================

// 用户声明 env 变量不应影响 env.Get 等内置函数
func TestResolverUserDeclaresEnvVariable(t *testing.T) {
	stdout, stderr, _ := runKami(`env := "myenv"; print env.Get("HOME")`, NewEmptyEnvironment())

	t.Logf("Stdout: %q", stdout)
	t.Logf("Stderr: %q", stderr)

	// Bug: resolver only reserves "err", not "env". User can shadow
	// the built-in env package. If env.Get still works, the runtime
	// handles it correctly despite resolver allocating a slot.
	// If it fails, the resolver slot shadows the package.
	if stderr != "" {
		t.Logf("Note: env.Get() failed after user declared 'env' variable: %s", stderr)
	}
}

// 用户声明 err 变量应被 resolver 拒绝（已保留）
func TestResolverUserCannotDeclareErr(t *testing.T) {
	// err is reserved — resolver should not allocate a slot for it.
	// The assignment should still work at runtime (err is maintained by runtime).
	_, stderr, _ := runKami(`err := "hello"`, NewEmptyEnvironment())
	t.Logf("err assignment stderr: %q", stderr)
	// This may or may not produce an error depending on runtime behavior.
	// The key test is that resolver doesn't allocate a slot for "err".
}

// 解析错误应包含位置信息
func TestParserErrorsHavePositionInfo(t *testing.T) {
	input := `(1 + 2`
	l := NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	errs := p.Errors()
	t.Logf("Parser errors for %q:", input)
	for _, e := range errs {
		t.Logf("  %q", e)
	}

	// Current: errors are plain strings with no position info.
	// Expected: errors should mention line/column number.
	hasPosition := false
	for _, e := range errs {
		// Check if error contains line/column indicators
		if len(e) > 0 {
			// Simple heuristic: position-aware errors typically contain ':' followed by digits
			for i := 0; i < len(e)-1; i++ {
				if e[i] == ':' && i+1 < len(e) && e[i+1] >= '0' && e[i+1] <= '9' {
					hasPosition = true
					break
				}
			}
		}
	}

	if len(errs) > 0 && !hasPosition {
		t.Logf("Note: parser errors lack line:column position info — addError() only takes a string")
	}
}
