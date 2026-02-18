package ast

import (
	"kamishell/internal/lexer"
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
	Token     lexer.Token
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

type PrintStatement struct {
	Token      lexer.Token
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
	Token lexer.Token // the := token
	Name  *Identifier
	Value Expression
}

func (as *AssignStatement) statementNode()       {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignStatement) String() string {
	var out strings.Builder
	out.WriteString(as.Name.String())
	out.WriteString(" := ")
	if as.Value != nil {
		out.WriteString(as.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return "\"" + sl.Value + "\"" }

type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

type BlockStatement struct {
	Token      lexer.Token // the { token
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
	Token       lexer.Token // the if token
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
	Token      lexer.Token // the first token of the expression
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
	Token      lexer.Token // the exec token
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

type InfixExpression struct {
	Token    lexer.Token // The operator token, e.g. +
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

type PipeStatement struct {
	Token    lexer.Token // The | token
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
	Token  lexer.Token // > or >>
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
	Token       lexer.Token // the for token
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
