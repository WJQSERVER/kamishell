package core

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
	return &Environment{store: s, types: t, packageStore: make(map[string]map[string]string)}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEmptyEnvironment()
	env.outer = outer
	if outer != nil {
		env.packageStore = outer.packageStore
	}
	return env
}

func NewScriptEnvironment(outer *Environment) *Environment {
	env := NewEmptyEnvironment()
	env.outer = outer
	env.packageStore = make(map[string]map[string]string)
	return env
}

type Environment struct {
	store        map[string]Object
	types        map[string]string
	outer        *Environment
	packageStore map[string]map[string]string
}

func (e *Environment) GetObject(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.GetObject(name)
	}
	return obj, ok
}

func (e *Environment) Get(name string) (interface{}, bool) {
	obj, ok := e.GetObject(name)
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
	obj, typeName, ok := normalizeValue(val)
	if !ok {
		return
	}
	e.store[name] = obj
	if shouldTrackType(typeName) {
		e.types[name] = typeName
	}
}

func (e *Environment) SetWithType(name string, val Object, typeName string) {
	e.store[name] = val
	if shouldTrackType(typeName) {
		e.types[name] = typeName
	}
}

func (e *Environment) Assign(name string, val Object) {
	if scope := e.scopeWithValue(name); scope != nil {
		scope.store[name] = val
		if _, ok := scope.types[name]; !ok && shouldTrackType(string(val.Type())) {
			scope.types[name] = string(val.Type())
		}
		return
	}
	e.Set(name, val)
}

func NewEmptyEnvironment() *Environment {
	return &Environment{
		store:        make(map[string]Object),
		types:        make(map[string]string),
		packageStore: make(map[string]map[string]string),
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

func (e *Environment) scopeWithValue(name string) *Environment {
	if _, ok := e.store[name]; ok {
		return e
	}
	if e.outer != nil {
		return e.outer.scopeWithValue(name)
	}
	return nil
}

func (e *Environment) SetPackageValue(pkg, name, value string) {
	if pkg == "" || name == "" {
		return
	}
	if e.packageStore == nil {
		e.packageStore = make(map[string]map[string]string)
	}
	if _, ok := e.packageStore[pkg]; !ok {
		e.packageStore[pkg] = make(map[string]string)
	}
	e.packageStore[pkg][name] = value
}

func (e *Environment) GetPackageValue(pkg, name string) (string, bool) {
	if e.packageStore == nil || pkg == "" || name == "" {
		return "", false
	}
	values, ok := e.packageStore[pkg]
	if !ok {
		return "", false
	}
	value, ok := values[name]
	return value, ok
}

func (e *Environment) DeletePackageValue(pkg, name string) bool {
	if e.packageStore == nil || pkg == "" || name == "" {
		return false
	}
	values, ok := e.packageStore[pkg]
	if !ok {
		return false
	}
	if _, ok := values[name]; !ok {
		return false
	}
	delete(values, name)
	return true
}

func (e *Environment) PackageSnapshot(pkg string) map[string]string {
	snapshot := make(map[string]string)
	if e.packageStore == nil || pkg == "" {
		return snapshot
	}
	values, ok := e.packageStore[pkg]
	if !ok {
		return snapshot
	}
	for key, value := range values {
		snapshot[key] = value
	}
	return snapshot
}

func normalizeValue(val interface{}) (Object, string, bool) {
	if obj, ok := val.(Object); ok {
		return obj, string(obj.Type()), true
	}
	if s, ok := val.(string); ok {
		return &String{Value: s}, string(STRING_OBJ), true
	}
	if i, ok := val.(int64); ok {
		return &Integer{Value: i}, string(INTEGER_OBJ), true
	}
	if b, ok := val.(bool); ok {
		if b {
			return TRUE, string(BOOLEAN_OBJ), true
		}
		return FALSE, string(BOOLEAN_OBJ), true
	}
	return nil, "", false
}

func shouldTrackType(typeName string) bool {
	return typeName != "" && typeName != string(NULL_OBJ)
}
