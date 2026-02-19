package main

import "os"
import "strings"

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	t := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		s[pair[0]] = &String{Value: pair[1]}
		t[pair[0]] = string(STRING_OBJ)
	}
	return &Environment{store: s, types: t}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEmptyEnvironment()
	env.outer = outer
	return env
}

type Environment struct {
	store map[string]Object
	types map[string]string
	outer *Environment
}

func (e *Environment) Get(name string) (interface{}, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) GetType(name string) (string, bool) {
	t, ok := e.types[name]
	if !ok && e.outer != nil {
		return e.outer.GetType(name)
	}
	return t, ok
}

func (e *Environment) Set(name string, val interface{}) {
	if obj, ok := val.(Object); ok {
		e.store[name] = obj
	} else if s, ok := val.(string); ok {
		e.store[name] = &String{Value: s}
	} else if i, ok := val.(int64); ok {
		e.store[name] = &Integer{Value: i}
	} else if b, ok := val.(bool); ok {
		if b {
			e.store[name] = TRUE
		} else {
			e.store[name] = FALSE
		}
	}
}

func (e *Environment) SetWithType(name string, val Object, typeName string) {
	e.store[name] = val
	if typeName != "" {
		e.types[name] = typeName
	}
}

func NewEmptyEnvironment() *Environment {
	return &Environment{
		store: make(map[string]Object),
		types: make(map[string]string),
	}
}

func (e *Environment) Keys() []string {
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	if e.outer != nil {
		keys = append(keys, e.outer.Keys()...)
	}
	return keys
}
