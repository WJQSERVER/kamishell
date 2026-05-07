package core

import (
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

type CommandStatement struct {
	Token     Token
	Name      string
	Arguments []Expression
}

func (cs *CommandStatement) statementNode()       {}
func (cs *CommandStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CommandStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.Name)
	for _, arg := range cs.Arguments {
		out.WriteString(" ")
		out.WriteString(arg.String())
	}
	out.WriteString(";")
	return out.String()
}

type InvalidStatement struct {
	Token   Token
	Message string
}

func (is *InvalidStatement) statementNode()       {}
func (is *InvalidStatement) TokenLiteral() string { return is.Token.Literal }
func (is *InvalidStatement) String() string       { return is.Message }

type PrintStatement struct {
	Token      Token
	Expression Expression
}

func (ps *PrintStatement) statementNode()       {}
func (ps *PrintStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PrintStatement) String() string {
	var out strings.Builder
	out.WriteString(ps.TokenLiteral())
	out.WriteString(" ")
	if ps.Expression != nil {
		out.WriteString(ps.Expression.String())
	}
	out.WriteString(";")
	return out.String()
}

type Parameter struct {
	Name     string
	TypeName string
}

func (p Parameter) String() string {
	if p.TypeName != "" {
		return p.Name + " " + p.TypeName
	}
	return p.Name
}

type AssignStatement struct {
	Token  Token // the := or = token
	Names  []string     // variable names: [x] for single, [x, y] for multi-assign
	Target Expression   // non-nil for index assignment: arr[i] = val
	Value  Expression
	// Resolver fields for = reassignment fast path
	ResolvedScopeDepth int // -1 = unresolved
	ResolvedSlotIndex  int // -1 = unresolved
}

func (as *AssignStatement) statementNode()       {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignStatement) String() string {
	var out strings.Builder
	out.WriteString(strings.Join(as.Names, ", "))
	out.WriteString(" := ")
	if as.Value != nil {
		out.WriteString(as.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type PointerAssignStatement struct {
	Token Token // the = token
	Target Expression // the *p expression
	Value  Expression
}

func (pas *PointerAssignStatement) statementNode()       {}
func (pas *PointerAssignStatement) TokenLiteral() string { return pas.Token.Literal }
func (pas *PointerAssignStatement) String() string {
	var out strings.Builder
	out.WriteString(pas.Target.String())
	out.WriteString(" = ")
	if pas.Value != nil {
		out.WriteString(pas.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type Identifier struct {
	Token      Token
	Value      string
	ScopeDepth int // -1 = unresolved; 0 = current scope, 1 = outer, etc.
	SlotIndex  int // -1 = unresolved; >= 0 = slot index in that scope
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// NewIdentifier creates an Identifier with unresolved scope defaults.
func NewIdentifier(tok Token, val string) *Identifier {
	return &Identifier{Token: tok, Value: val, ScopeDepth: -1, SlotIndex: -1}
}

type StringLiteral struct {
	Token Token
	Value string
	Obj   *String
	Parts []StringPart // pre-parsed interpolation segments (nil = no $)
}

// StringPart represents a segment of a string literal.
// If Var is non-empty, it's a variable reference; otherwise Text is a literal.
type StringPart struct {
	Text string // literal text (empty if Var is set)
	Var  string // variable name (empty if Text is set)
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

type IntegerLiteral struct {
	Token Token
	Value int64
	Obj   *Integer
	Err   string
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type FloatLiteral struct {
	Token Token
	Value float64
	Obj   *Float
	Err   string
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

type BooleanLiteral struct {
	Token Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

type ArrayLiteral struct {
	Token    Token // the [ token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out strings.Builder
	out.WriteString("[")
	for i, el := range al.Elements {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(el.String())
	}
	out.WriteString("]")
	return out.String()
}

type IndexExpression struct {
	Token Token // the [ token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}

type BlockStatement struct {
	Token      Token // the { token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out strings.Builder
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

type IfStatement struct {
	Token       Token // the if token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if ")
	out.WriteString(is.Condition.String())
	out.WriteString(" { ")
	out.WriteString(is.Consequence.String())
	out.WriteString(" }")
	if is.Alternative != nil {
		out.WriteString(" else { ")
		out.WriteString(is.Alternative.String())
		out.WriteString(" }")
	}
	return out.String()
}

type ExpressionStatement struct {
	Token      Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type ExecStatement struct {
	Token      Token // the exec token
	CommandStr Expression
}

func (es *ExecStatement) statementNode()       {}
func (es *ExecStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExecStatement) String() string {
	var out strings.Builder
	out.WriteString(es.TokenLiteral())
	out.WriteString(" ")
	if es.CommandStr != nil {
		out.WriteString(es.CommandStr.String())
	}
	out.WriteString(";")
	return out.String()
}

type PrefixExpression struct {
	Token    Token // The operator token, e.g. & or *
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

type InfixExpression struct {
	Token    Token // The operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")
	return out.String()
}

type CallExpression struct {
	Token     Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out strings.Builder
	args := make([]string, 0, len(ce.Arguments))
	for _, arg := range ce.Arguments {
		args = append(args, arg.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

type MemberExpression struct {
	Token    Token
	Object   Expression
	Property string
}

func (me *MemberExpression) expressionNode()      {}
func (me *MemberExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MemberExpression) String() string {
	return me.Object.String() + "." + me.Property
}

type PipeStatement struct {
	Token    Token       // The | token
	Commands []Statement // The commands in the pipeline (usually CommandStatements)
}

func (ps *PipeStatement) statementNode()       {}
func (ps *PipeStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PipeStatement) String() string {
	var out strings.Builder
	for i, cmd := range ps.Commands {
		out.WriteString(cmd.String())
		if i < len(ps.Commands)-1 {
			out.WriteString(" | ")
		}
	}
	return out.String()
}

type RedirectStatement struct {
	Token  Token // > or >>
	Source Statement
	Target Expression
	Append bool
}

func (rs *RedirectStatement) statementNode()       {}
func (rs *RedirectStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *RedirectStatement) String() string {
	var out strings.Builder
	out.WriteString(rs.Source.String())
	out.WriteString(" ")
	out.WriteString(rs.Token.Literal)
	out.WriteString(" ")
	out.WriteString(rs.Target.String())
	return out.String()
}

type ForStatement struct {
	Token       Token // the for token
	Init        Statement  // init statement (3-clause: i := 0), nil for while-style
	Condition   Expression // condition expression, nil for infinite loop
	Post        Statement  // post statement (3-clause: i = i + 1), nil for while-style
	Consequence *BlockStatement
	// Pre-analyzed increment pattern (Parser stage)
	IncVarName string // variable name for i = i + N pattern
	IncDelta   int64  // +1 or -1 for i = i +/- 1
	HasInc     bool   // true if body is a single i = i +/- 1 assignment
	// Resolver fields for increment fast path
	IncScopeDepth int // -1 = unresolved
	IncSlotIndex  int // -1 = unresolved
	// Iterator range (for v := range iter(args) { ... })
	IsIterRange bool       // true if this is a range-over-function
	IterCall    Expression // the iterator call expression (e.g. iter(args))
	IterVars    []string   // variable names: ["v"] or ["k", "v"]
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for ")
	if fs.Init != nil {
		out.WriteString(fs.Init.String())
		out.WriteString(" ")
	}
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	if fs.Post != nil {
		out.WriteString("; ")
		out.WriteString(fs.Post.String())
	}
	out.WriteString(" { ")
	if fs.Consequence != nil {
		out.WriteString(fs.Consequence.String())
	}
	out.WriteString(" }")
	return out.String()
}

type SwitchStatement struct {
	Token        Token       // the switch token
	Tag          Expression  // optional, nil for tagless switch
	Cases        []CaseClause
	IntSwitch    bool // true if all non-default case values are integer literals
	StringSwitch bool // true if all non-default case values are string literals (no $)
}

func (ss *SwitchStatement) statementNode()       {}
func (ss *SwitchStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SwitchStatement) String() string {
	var out strings.Builder
	out.WriteString("switch ")
	if ss.Tag != nil {
		out.WriteString(ss.Tag.String())
	}
	out.WriteString(" { ")
	for _, c := range ss.Cases {
		out.WriteString(c.String())
	}
	out.WriteString(" }")
	return out.String()
}

type CaseClause struct {
	Token  Token          // case or default keyword
	Values []Expression   // case values; nil for default
	Body   *BlockStatement
	IntConsts    []int64  // pre-computed integer literal values (Parser stage)
	StringConsts []string // pre-computed string literal values (Parser stage)
	HasConstVals bool     // true when all Values are same-type literals
}

func (cc *CaseClause) String() string {
	var out strings.Builder
	if cc.Values == nil {
		out.WriteString("default:")
	} else {
		out.WriteString("case ")
		for i, v := range cc.Values {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(v.String())
		}
		out.WriteString(":")
	}
	if cc.Body != nil {
		out.WriteString(" ")
		out.WriteString(cc.Body.String())
	}
	return out.String()
}

type BreakStatement struct {
	Token Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

type ContinueStatement struct {
	Token Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue" }

type LogicalStatement struct {
	Token    Token // && or ||
	Left     Statement
	Operator string
	Right    Statement
}

func (ls *LogicalStatement) statementNode()       {}
func (ls *LogicalStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LogicalStatement) String() string {
	var out strings.Builder
	out.WriteString("(")
	out.WriteString(ls.Left.String())
	out.WriteString(" " + ls.Operator + " ")
	out.WriteString(ls.Right.String())
	out.WriteString(")")
	return out.String()
}

type FunctionStatement struct {
	Token       Token // the func token
	Name        string
	Parameters  []Parameter
	ReturnTypes []string // empty = void, len=1 single, len>1 multi-return
	Body        *BlockStatement
	Obj         *Function
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	var out strings.Builder
	out.WriteString(fs.TokenLiteral() + " ")
	out.WriteString(fs.Name)
	out.WriteString("(")
	params := make([]string, len(fs.Parameters))
	for i, p := range fs.Parameters {
		params[i] = p.String()
	}
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	if len(fs.ReturnTypes) > 0 {
		out.WriteString(" " + strings.Join(fs.ReturnTypes, ", "))
	}
	out.WriteString(" ")
	out.WriteString(fs.Body.String())
	return out.String()
}

type FunctionLiteral struct {
	Token       Token // the func token
	Parameters  []Parameter
	ReturnTypes []string
	Body        *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out strings.Builder
	out.WriteString("func(")
	params := make([]string, len(fl.Parameters))
	for i, p := range fl.Parameters {
		params[i] = p.String()
	}
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	if len(fl.ReturnTypes) > 0 {
		out.WriteString(" " + strings.Join(fl.ReturnTypes, ", "))
	}
	out.WriteString(" ")
	out.WriteString(fl.Body.String())
	return out.String()
}

type NilLiteral struct {
	Token Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return nl.Token.Literal }

type BackgroundStatement struct {
	Token Token // the & token
	Stmt  Statement
}

func (bs *BackgroundStatement) statementNode()       {}
func (bs *BackgroundStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BackgroundStatement) String() string {
	var out strings.Builder
	out.WriteString(bs.Stmt.String())
	out.WriteString(" &")
	return out.String()
}

type GoStatement struct {
	Token Token // the go token
	Node  Node  // BlockStatement or CommandStatement or Function call
}

func (gs *GoStatement) statementNode()       {}
func (gs *GoStatement) TokenLiteral() string { return gs.Token.Literal }
func (gs *GoStatement) String() string {
	var out strings.Builder
	out.WriteString("go ")
	out.WriteString(gs.Node.String())
	return out.String()
}

type GoExpression struct {
	Token Token // the go token
	Node  Node  // BlockStatement or CommandStatement or Function call
}

func (ge *GoExpression) expressionNode()      {}
func (ge *GoExpression) TokenLiteral() string { return ge.Token.Literal }
func (ge *GoExpression) String() string {
	var out strings.Builder
	out.WriteString("go ")
	out.WriteString(ge.Node.String())
	return out.String()
}

type ReturnStatement struct {
	Token        Token // the return token
	ReturnValues []Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString(rs.TokenLiteral() + " ")
	vals := make([]string, len(rs.ReturnValues))
	for i, v := range rs.ReturnValues {
		vals[i] = v.String()
	}
	out.WriteString(strings.Join(vals, ", "))
	out.WriteString(";")
	return out.String()
}

type VarStatement struct {
	Token    Token // the var token
	Name     string
	TypeName string
	Value    Expression // optional initial value
}

func (vs *VarStatement) statementNode()       {}
func (vs *VarStatement) TokenLiteral() string { return vs.Token.Literal }
func (vs *VarStatement) String() string {
	var out strings.Builder
	out.WriteString("var ")
	out.WriteString(vs.Name)
	if vs.TypeName != "" {
		out.WriteString(" ")
		out.WriteString(vs.TypeName)
	}
	if vs.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type ImportStatement struct {
	Token   Token // the import token
	Path    string // import path like "Go" or "Go/fmt"
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	var out strings.Builder
	out.WriteString("import \"")
	out.WriteString(is.Path)
	out.WriteString("\"")
	return out.String()
}

type MethodCallBlockStatement struct {
	Token  Token           // the object token (e.g., "wg")
	Object Expression      // the object expression (e.g., Identifier{Value: "wg"})
	Method string          // the method name (e.g., "Go")
	Body   *BlockStatement // the block body
}

func (mcb *MethodCallBlockStatement) statementNode()       {}
func (mcb *MethodCallBlockStatement) TokenLiteral() string { return mcb.Token.Literal }
func (mcb *MethodCallBlockStatement) String() string {
	var out strings.Builder
	out.WriteString(mcb.Object.String())
	out.WriteString(".")
	out.WriteString(mcb.Method)
	out.WriteString(" ")
	out.WriteString(mcb.Body.String())
	return out.String()
}

type WaitStatement struct {
	Token   Token // the wait token
	Timeout Expression // optional timeout in seconds
}

func (ws *WaitStatement) statementNode()       {}
func (ws *WaitStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WaitStatement) String() string {
	var out strings.Builder
	out.WriteString("wait")
	if ws.Timeout != nil {
		out.WriteString("(")
		out.WriteString(ws.Timeout.String())
		out.WriteString(")")
	}
	return out.String()
}
