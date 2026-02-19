package main

import "fmt"

type ObjectType string

const (
	INTEGER_OBJ  ObjectType = "INTEGER"
	BOOLEAN_OBJ  ObjectType = "BOOLEAN"
	STRING_OBJ   ObjectType = "STRING"
	NULL_OBJ     ObjectType = "NULL"
	ERROR_OBJ    ObjectType = "ERROR"
	FUNCTION_OBJ ObjectType = "FUNCTION"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

type Boolean struct {
	Value bool
}

func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

type String struct {
	Value string
}

func (s *String) Inspect() string  { return s.Value }
func (s *String) Type() ObjectType { return STRING_OBJ }

type Null struct{}

func (n *Null) Inspect() string  { return "nil" }
func (n *Null) Type() ObjectType { return NULL_OBJ }

type Error struct {
	Message string
	Code    int
	Op      string
}

func (e *Error) Inspect() string {
	if e.Op != "" {
		return fmt.Sprintf("ERROR (%s): %s (code: %d)", e.Op, e.Message, e.Code)
	}
	return "ERROR: " + e.Message
}
func (e *Error) Type() ObjectType { return ERROR_OBJ }

type Function struct {
	Parameters []*Identifier
	Body       *BlockStatement
	Env        *Environment
}

func (f *Function) Inspect() string  { return "func" }
func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
