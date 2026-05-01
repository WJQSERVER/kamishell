package core

import "maps"

import "os"
import "strings"

func NewEnvironment() *Environment {
	environ := os.Environ()
	s := make(map[string]Object, len(environ))
	t := make(map[string]string, len(environ))
	for _, e := range environ {
		key, value, _ := strings.Cut(e, "=")
		s[key] = &String{Value: value}
		t[key] = string(STRING_OBJ)
	}
	return &Environment{store: s, types: t}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{store: make(map[string]Object), types: make(map[string]string), outer: outer, packageStore: outerPackageStore(outer)}
}

func NewScriptEnvironment(outer *Environment) *Environment {
	return &Environment{store: make(map[string]Object), types: make(map[string]string), outer: outer}
}

func NewFunctionCallEnvironment(outer *Environment, paramCapacity int) *Environment {
	storeCap := paramCapacity
	if storeCap < 1 {
		storeCap = 1
	}
	return &Environment{
		store:        make(map[string]Object, storeCap),
		types:        make(map[string]string, storeCap),
		outer:        outer,
		packageStore: outerPackageStore(outer),
	}
}

type Environment struct {
	store        map[string]Object
	types        map[string]string
	outer        *Environment
	packageStore map[string]map[string]string
}

func (e *Environment) Clone() *Environment {
	if e == nil {
		return nil
	}
	clone := &Environment{
		store:        make(map[string]Object, len(e.store)),
		types:        make(map[string]string, len(e.types)),
		packageStore: clonePackageStore(e.packageStore),
	}
	if len(e.types) > 0 {
		maps.Copy(clone.types, e.types)
	}
	if e.outer != nil {
		clone.outer = e.outer.Clone()
	}
	for key, value := range e.store {
		clone.store[key] = cloneObjectForEnv(value, clone)
	}
	return clone
}

func (e *Environment) GetObject(name string) (Object, bool) {
	for scope := e; scope != nil; scope = scope.outer {
		if obj, ok := scope.store[name]; ok {
			return obj, true
		}
	}
	return nil, false
}

func (e *Environment) Get(name string) (any, bool) {
	for scope := e; scope != nil; scope = scope.outer {
		if obj, ok := scope.store[name]; ok {
			return obj, true
		}
	}
	return nil, false
}

func (e *Environment) GetType(name string) (string, bool) {
	for scope := e; scope != nil; scope = scope.outer {
		if typeName, ok := scope.types[name]; ok {
			return typeName, true
		}
	}
	return "", false
}

func (e *Environment) Set(name string, val any) {
	obj, typeName, ok := normalizeValue(val)
	if !ok {
		return
	}
	e.store[name] = obj
	if shouldTrackType(typeName) {
		e.ensureTypes()
		e.types[name] = typeName
	}
}

func (e *Environment) SetWithType(name string, val Object, typeName string) {
	e.store[name] = val
	if shouldTrackType(typeName) {
		e.ensureTypes()
		e.types[name] = typeName
	}
}

func (e *Environment) SetObject(name string, val Object) {
	e.store[name] = val
	if val != nil {
		typeName := string(val.Type())
		if shouldTrackType(typeName) {
			e.ensureTypes()
			e.types[name] = typeName
		}
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

func (e *Environment) ResolveForAssign(name string) (*Environment, string, bool) {
	for scope := e; scope != nil; scope = scope.outer {
		if _, ok := scope.store[name]; ok {
			typeName, hasType := scope.types[name]
			return scope, typeName, hasType
		}
	}
	return nil, "", false
}

func NewEmptyEnvironment() *Environment {
	return &Environment{store: make(map[string]Object), types: make(map[string]string)}
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
	for scope := e; scope != nil; scope = scope.outer {
		if _, ok := scope.store[name]; ok {
			return scope
		}
	}
	return nil
}

func (e *Environment) SetPackageValue(pkg, name, value string) {
	if pkg == "" || name == "" {
		return
	}
	e.ensurePackageStore()
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
	maps.Copy(snapshot, values)
	return snapshot
}

func normalizeValue(val any) (Object, string, bool) {
	if obj, ok := val.(Object); ok {
		return obj, string(obj.Type()), true
	}
	if s, ok := val.(string); ok {
		return &String{Value: s}, string(STRING_OBJ), true
	}
	if i, ok := val.(int64); ok {
		return getIntegerObject(i), string(INTEGER_OBJ), true
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
	return typeName != ""
}

func clonePackageStore(src map[string]map[string]string) map[string]map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]map[string]string, len(src))
	for pkg, values := range src {
		inner := make(map[string]string, len(values))
		maps.Copy(inner, values)
		dst[pkg] = inner
	}
	return dst
}

func cloneObjectForEnv(obj Object, owner *Environment) Object {
	fn, ok := obj.(*Function)
	if !ok {
		return obj
	}
	cloned := *fn
	cloned.Env = owner
	return &cloned
}

func outerPackageStore(outer *Environment) map[string]map[string]string {
	if outer == nil {
		return nil
	}
	return outer.packageStore
}

func (e *Environment) ensurePackageStore() {
	if e.packageStore == nil {
		e.packageStore = make(map[string]map[string]string)
	}
}

func (e *Environment) ensureTypes() {
	if e.types == nil {
		e.types = make(map[string]string)
	}
}
