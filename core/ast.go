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

type AssignStatement struct {
	Token Token // the := token
	Name  string
	Value Expression
}

func (as *AssignStatement) statementNode()       {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignStatement) String() string {
	var out strings.Builder
	out.WriteString(as.Name)
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
	Token Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type StringLiteral struct {
	Token Token
	Value string
	Obj   *String
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
	Condition   Expression
	Consequence *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for ")
	if fs.Condition != nil {
		out.WriteString(fs.Condition.String())
	}
	out.WriteString(" { ")
	out.WriteString(fs.Consequence.String())
	out.WriteString(" }")
	return out.String()
}

type SwitchStatement struct {
	Token Token       // the switch token
	Tag   Expression  // optional, nil for tagless switch
	Cases []CaseClause
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
	Token      Token // the func token
	Name       string
	Parameters []string
	Body       *BlockStatement
	Obj        *Function
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	var out strings.Builder
	out.WriteString(fs.TokenLiteral() + " ")
	out.WriteString(fs.Name)
	out.WriteString("(")
	out.WriteString(strings.Join(fs.Parameters, ", "))
	out.WriteString(") ")
	out.WriteString(fs.Body.String())
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
	Token       Token // the return token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
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
