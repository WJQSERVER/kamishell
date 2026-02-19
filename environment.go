package kamishell

import "os"
import "strings"

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		s[pair[0]] = &String{Value: pair[1]}
	}
	return &Environment{store: s}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEmptyEnvironment()
	env.outer = outer
	return env
}

type Environment struct {
	store map[string]Object
	outer *Environment
}

func (e *Environment) Get(name string) (interface{}, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
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

func NewEmptyEnvironment() *Environment {
	return &Environment{store: make(map[string]Object)}
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
