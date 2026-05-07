package core

import "kamishell/builtin"

// reservedNames are names that the Resolver must NOT assign slots to,
// because they are maintained by the runtime via SetObject/SetWithType
// outside of the normal declaration flow.
var reservedNames = map[string]bool{
	"err": true,
}

// Resolve performs a variable resolution pass on the parsed AST.
// It walks the tree, tracking declarations and scope depth, and marks
// each Identifier with the scope depth and slot index where its value lives.
// At runtime, resolved identifiers use O(1) slice indexing instead of O(depth) map lookups.
func Resolve(program *Program) {
	r := &resolver{
		scopes: []scope{{nameToSlot: make(map[string]int)}},
	}
	for _, stmt := range program.Statements {
		r.resolveStatement(stmt)
	}
}

type scope struct {
	nameToSlot map[string]int // name → slot index in this scope
	slotCount  int            // next available slot index
}

type resolver struct {
	scopes []scope
}

func (r *resolver) currentScope() *scope {
	return &r.scopes[len(r.scopes)-1]
}

func (r *resolver) pushScope() {
	r.scopes = append(r.scopes, scope{nameToSlot: make(map[string]int)})
}

func (r *resolver) popScope() {
	r.scopes = r.scopes[:len(r.scopes)-1]
}

// declare allocates a slot for name in the current scope and returns its index.
func (r *resolver) declare(name string) int {
	s := r.currentScope()
	idx := s.slotCount
	s.nameToSlot[name] = idx
	s.slotCount++
	return idx
}

// resolve looks up name through the scope chain and returns (depth, slotIndex).
// Returns (-1, -1) if not found.
func (r *resolver) resolve(name string) (depth, slot int) {
	for i := len(r.scopes) - 1; i >= 0; i-- {
		if idx, ok := r.scopes[i].nameToSlot[name]; ok {
			return len(r.scopes) - 1 - i, idx
		}
	}
	return -1, -1
}

func (r *resolver) resolveStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *AssignStatement:
		r.resolveAssignStatement(s)
	case *VarStatement:
		r.resolveVarStatement(s)
	case *FunctionStatement:
		r.resolveFunctionStatement(s)
	case *ExpressionStatement:
		r.resolveExpression(s.Expression)
	case *PrintStatement:
		r.resolveExpression(s.Expression)
	case *IfStatement:
		r.resolveIfStatement(s)
	case *ForStatement:
		r.resolveForStatement(s)
	case *BlockStatement:
		r.resolveBlockStatement(s)
	case *ReturnStatement:
		r.resolveReturnStatement(s)
	case *SwitchStatement:
		r.resolveSwitchStatement(s)
	case *PipeStatement:
		r.resolvePipeStatement(s)
	case *RedirectStatement:
		r.resolveRedirectStatement(s)
	case *LogicalStatement:
		r.resolveStatement(s.Left)
		r.resolveStatement(s.Right)
	case *BackgroundStatement:
		r.resolveStatement(s.Stmt)
	case *GoStatement:
		r.resolveGoNode(s.Node)
	case *ExecStatement:
		r.resolveExpression(s.CommandStr)
	case *ImportStatement:
		// no-op
	case *WaitStatement:
		r.resolveExpression(s.Timeout)
	case *MethodCallBlockStatement:
		r.resolveExpression(s.Object)
		r.resolveBlockStatement(s.Body)
	case *PointerAssignStatement:
		// Do NOT resolve the target (*p) — pointers use map-based EnvEntry
		r.resolveExpression(s.Value)
	case *BreakStatement, *ContinueStatement:
		// no-op
	case *InvalidStatement:
		// no-op
	}
}

func (r *resolver) resolveAssignStatement(s *AssignStatement) {
	if s.Target != nil {
		// Index assignment: arr[i] = val — resolve the value, not the target
		r.resolveExpression(s.Value)
		return
	}

	if s.Token.Literal == ":=" {
		// Short variable declaration: declare in current scope
		for _, name := range s.Names {
			if !reservedNames[name] {
				r.declare(name)
			}
		}
		r.resolveExpression(s.Value)
	} else {
		// Reassignment: resolve the target variable
		if len(s.Names) == 1 {
			name := s.Names[0]
			if !reservedNames[name] {
				depth, slot := r.resolve(name)
				s.ResolvedScopeDepth = depth
				s.ResolvedSlotIndex = slot
			}
		}
		r.resolveExpression(s.Value)
	}
}

func (r *resolver) resolveVarStatement(s *VarStatement) {
	if !reservedNames[s.Name] {
		r.declare(s.Name)
	}
	r.resolveExpression(s.Value)
}

func (r *resolver) resolveFunctionStatement(s *FunctionStatement) {
	// Declare the function name as a constant in the current scope
	if !reservedNames[s.Name] {
		r.declare(s.Name)
	}

	// Resolve the function body in a new scope with parameters
	r.pushScope()
	// Parameters get slots 0, 1, 2, ...
	for _, param := range s.Parameters {
		if !reservedNames[param.Name] {
			r.declare(param.Name)
		}
	}
	if s.Body != nil {
		r.resolveBlockStatementInner(s.Body)
	}

	// Record slot capacity on the function object for runtime allocation
	if s.Obj != nil {
		s.Obj.SlotCapacity = r.currentScope().slotCount
	} else {
		// Store capacity on the statement for later use when creating Function objects
		// We'll use a temporary field — actually, let's just set it on the Function
		// when evalFunctionStatement creates it. For now, record on the statement.
		// We'll use the FunctionStatement to carry this info.
	}
	r.popScope()
}

func (r *resolver) resolveIfStatement(s *IfStatement) {
	r.resolveExpression(s.Condition)
	if s.Consequence != nil {
		r.resolveBlockStatement(s.Consequence)
	}
	if s.Alternative != nil {
		r.resolveBlockStatement(s.Alternative)
	}
}

func (r *resolver) resolveForStatement(s *ForStatement) {
	// For-loop variables live in the enclosing scope (same as Go).
	// Do NOT push a new scope here — the runtime doesn't create a new Environment for for-loops.

	// Resolve init (may declare variables like i := 0)
	if s.Init != nil {
		r.resolveStatement(s.Init)
	}

	// Resolve condition
	r.resolveExpression(s.Condition)

	// Resolve post
	if s.Post != nil {
		r.resolveStatement(s.Post)
	}

	// Resolve body — use resolveBlockStatementInner (no new scope push)
	// because evalLoopBody evaluates in the same env as the for-loop
	if s.Consequence != nil {
		r.resolveBlockStatementInner(s.Consequence)
	}

	// Resolve iterator range variables
	if s.IsIterRange {
		for _, v := range s.IterVars {
			if !reservedNames[v] {
				r.declare(v)
			}
		}
	}

	// If this for loop has an increment pattern, resolve the variable
	if s.HasInc && s.IncVarName != "" {
		depth, slot := r.resolve(s.IncVarName)
		s.IncScopeDepth = depth
		s.IncSlotIndex = slot
	}
}

func (r *resolver) resolveBlockStatement(s *BlockStatement) {
	r.pushScope()
	r.resolveBlockStatementInner(s)
	r.popScope()
}

func (r *resolver) resolveBlockStatementInner(s *BlockStatement) {
	if s == nil {
		return
	}
	for _, stmt := range s.Statements {
		r.resolveStatement(stmt)
	}
}

func (r *resolver) resolveReturnStatement(s *ReturnStatement) {
	for _, rv := range s.ReturnValues {
		r.resolveExpression(rv)
	}
}

func (r *resolver) resolveSwitchStatement(s *SwitchStatement) {
	r.resolveExpression(s.Tag)
	for i := range s.Cases {
		c := &s.Cases[i]
		for _, v := range c.Values {
			r.resolveExpression(v)
		}
		if c.Body != nil {
			r.resolveBlockStatement(c.Body)
		}
	}
}

func (r *resolver) resolvePipeStatement(s *PipeStatement) {
	for _, cmd := range s.Commands {
		r.resolveStatement(cmd)
	}
}

func (r *resolver) resolveRedirectStatement(s *RedirectStatement) {
	r.resolveStatement(s.Source)
	r.resolveExpression(s.Target)
}

func (r *resolver) resolveGoNode(node Node) {
	switch n := node.(type) {
	case *BlockStatement:
		// go { ... } — resolve in a new scope (the env will be cloned at runtime)
		r.resolveBlockStatement(n)
	case *ExpressionStatement:
		r.resolveExpression(n.Expression)
	case *CommandStatement:
		// Command names are not resolved as variables
		for _, arg := range n.Arguments {
			r.resolveExpression(arg)
		}
	}
}

func (r *resolver) resolveExpression(expr Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *Identifier:
		r.resolveIdentifier(e)
	case *InfixExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Right)
	case *PrefixExpression:
		if e.Operator == "&" {
			// Address-of: do NOT resolve the operand — pointers use map-based EnvEntry
			return
		}
		r.resolveExpression(e.Right)
	case *CallExpression:
		r.resolveExpression(e.Function)
		for _, arg := range e.Arguments {
			r.resolveExpression(arg)
		}
	case *MemberExpression:
		r.resolveExpression(e.Object)
		// Property is not a variable — don't resolve
	case *IndexExpression:
		r.resolveExpression(e.Left)
		r.resolveExpression(e.Index)
	case *ArrayLiteral:
		for _, el := range e.Elements {
			r.resolveExpression(el)
		}
	case *FunctionLiteral:
		// Anonymous function: resolve in a new scope
		r.pushScope()
		for _, param := range e.Parameters {
			if !reservedNames[param.Name] {
				r.declare(param.Name)
			}
		}
		if e.Body != nil {
			r.resolveBlockStatementInner(e.Body)
		}
		r.popScope()
	case *GoExpression:
		r.resolveGoNode(e.Node)
	case *StringLiteral:
		// String interpolation ($var) is handled at runtime via os.Expand,
		// which calls env.GetObject. The Identifier nodes for $var are NOT
		// in the AST — they're parsed at string expansion time. So no resolution here.
	case *IntegerLiteral, *FloatLiteral, *BooleanLiteral, *NilLiteral:
		// no-op
	}
}

func (r *resolver) resolveIdentifier(ident *Identifier) {
	// Skip reserved names
	if reservedNames[ident.Value] {
		return
	}

	// Skip package names — these return Package objects, not from store
	if ident.Value == "env" || ident.Value == "sync" || ident.Value == "param" {
		return
	}

	// Skip NativeFns — these are resolved before variable lookup
	if _, ok := NativeFns[ident.Value]; ok {
		return
	}

	// Skip builtin commands
	if _, ok := builtin.Builtins[ident.Value]; ok {
		return
	}

	// Try to resolve through scope chain
	depth, slot := r.resolve(ident.Value)
	if depth >= 0 {
		ident.ScopeDepth = depth
		ident.SlotIndex = slot
	}
}
