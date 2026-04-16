package core

import "strconv"

type ObjectType string

const (
	INTEGER_OBJ  ObjectType = "INTEGER"
	FLOAT_OBJ    ObjectType = "FLOAT"
	BOOLEAN_OBJ  ObjectType = "BOOLEAN"
	STRING_OBJ   ObjectType = "STRING"
	NULL_OBJ     ObjectType = "NULL"
	ERROR_OBJ    ObjectType = "ERROR"
	FUNCTION_OBJ ObjectType = "FUNCTION"
	PACKAGE_OBJ  ObjectType = "PACKAGE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Inspect() string  { return strconv.FormatInt(i.Value, 10) }
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

const (
	integerCacheMin int64 = -128
	integerCacheMax int64 = 1024
)

var integerCache = initIntegerCache()

func initIntegerCache() []*Integer {
	size := integerCacheMax - integerCacheMin + 1
	cache := make([]*Integer, size)
	for i := range cache {
		cache[i] = &Integer{Value: integerCacheMin + int64(i)}
	}
	return cache
}

func getIntegerObject(value int64) *Integer {
	if value >= integerCacheMin && value <= integerCacheMax {
		return integerCache[value-integerCacheMin]
	}
	return &Integer{Value: value}
}

func GetInteger(value int64) *Integer {
	return getIntegerObject(value)
}

type Float struct {
	Value float64
}

func (f *Float) Inspect() string  { return strconv.FormatFloat(f.Value, 'f', -1, 64) }
func (f *Float) Type() ObjectType { return FLOAT_OBJ }

type Boolean struct {
	Value bool
}

func (b *Boolean) Inspect() string {
	if b.Value {
		return "true"
	}
	return "false"
}
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
		return "ERROR (" + e.Op + "): " + e.Message + " (code: " + strconv.Itoa(e.Code) + ")"
	}
	return "ERROR: " + e.Message
}
func (e *Error) Type() ObjectType { return ERROR_OBJ }

type Function struct {
	Parameters []string
	Body       *BlockStatement
	Env        *Environment
}

func (f *Function) Inspect() string  { return "func" }
func (f *Function) Type() ObjectType { return FUNCTION_OBJ }

type NativeFunction struct {
	Fn func(env *Environment, args ...Object) Object
}

func (nf *NativeFunction) Type() ObjectType { return "NATIVE_FUNCTION" }
func (nf *NativeFunction) Inspect() string  { return "native function" }

type Package struct {
	Name string
}

func (p *Package) Type() ObjectType { return PACKAGE_OBJ }
func (p *Package) Inspect() string  { return p.Name }
