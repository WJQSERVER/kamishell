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
	Token     lexer.Token // the first token (e.g. ls)
	Name      string
	Arguments []string
}

func (cs *CommandStatement) statementNode()       {}
func (cs *CommandStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CommandStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.Name)
	for _, arg := range cs.Arguments {
		out.WriteString(" ")
		out.WriteString(arg)
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

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }
