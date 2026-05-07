package core

import "maps"

import "os"
import "strings"
import "sync"
import "unsafe"

// EnvEntry holds a variable's value for pointer reference support.
type EnvEntry struct {
	Owner *Environment
	Name  string
	Value Object
}

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
	return &Environment{store: make(map[string]Object), outer: outer, packageStore: outerPackageStore(outer)}
}

func NewScriptEnvironment(outer *Environment) *Environment {
	return &Environment{store: make(map[string]Object), outer: outer}
}

func NewFunctionCallEnvironment(outer *Environment, paramCapacity int) *Environment {
	storeCap := max(paramCapacity, 1)
	env := &Environment{
		store:        make(map[string]Object, storeCap),
		outer:        outer,
		packageStore: outerPackageStore(outer),
	}
	if paramCapacity > 0 {
		env.slots = make([]Object, 0, paramCapacity)
		env.slotByName = make(map[string]int, paramCapacity)
	}
	return env
}

const intFreelistCap = 16

type Environment struct {
	store        map[string]Object
	refStore     map[string]*EnvEntry // pointer reference storage (lazy)
	types        map[string]string
	constants    map[string]bool // constant names (from func declarations)
	outer        *Environment
	packageStore map[string]map[string]string
	intFreelist  []*Integer // recycled Integer objects (CPython-style freelist)
	freelistMu   sync.Mutex // protects intFreelist
	// Slot-based fast storage for resolved variables
	slots      []Object         // indexed storage (fast path)
	slotByName map[string]int   // name → slot index (for SetObject sync)
}

func (e *Environment) Clone() *Environment {
	if e == nil {
		return nil
	}
	clone := &Environment{
		store:        make(map[string]Object, len(e.store)),
		packageStore: clonePackageStore(e.packageStore),
	}
	if len(e.types) > 0 {
		clone.types = make(map[string]string, len(e.types))
		maps.Copy(clone.types, e.types)
	}
	if e.outer != nil {
		clone.outer = e.outer.Clone()
	}
	for key, value := range e.store {
		clone.store[key] = cloneObjectForEnv(value, clone)
	}
	// Copy slots
	if e.slots != nil {
		clone.slots = make([]Object, len(e.slots))
		copy(clone.slots, e.slots)
	}
	if e.slotByName != nil {
		clone.slotByName = make(map[string]int, len(e.slotByName))
		maps.Copy(clone.slotByName, e.slotByName)
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

func (e *Environment) SetString(name string, val string) {
	e.store[name] = &String{Value: val}
	if e.types != nil {
		e.types[name] = string(STRING_OBJ)
	}
}

func (e *Environment) GetString(name string) (string, bool) {
	for scope := e; scope != nil; scope = scope.outer {
		if obj, ok := scope.store[name]; ok {
			if s, ok := obj.(*String); ok {
				return s.Value, true
			}
			return obj.Inspect(), true
		}
	}
	return "", false
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
	// Sync refStore if entry exists
	if e.refStore != nil {
		if ref, ok := e.refStore[name]; ok {
			ref.Value = val
		}
	}
	// Sync slot if this variable has one
	e.SetSlotByName(name, val)
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
			scope.ensureTypes()
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

// MarkConstant marks a name as a constant (from func declarations).
// Constants cannot be reassigned via =.
func (e *Environment) MarkConstant(name string) {
	if e.constants == nil {
		e.constants = make(map[string]bool)
	}
	e.constants[name] = true
}

// IsConstant checks if a name is a constant.
func (e *Environment) IsConstant(name string) bool {
	if e.constants == nil {
		return false
	}
	return e.constants[name]
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
	if e.packageStore == nil || pkg == "" {
		return make(map[string]string)
	}
	values, ok := e.packageStore[pkg]
	if !ok {
		return make(map[string]string)
	}
	snapshot := make(map[string]string, len(values))
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
	return typeName != "" && typeName != string(NULL_OBJ)
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

// GetRef returns an EnvEntry for pointer operations.
// Creates the entry in refStore if it doesn't exist.
func (e *Environment) GetRef(name string) (*EnvEntry, bool) {
	// Check if variable exists in store
	if _, ok := e.store[name]; !ok {
		// Check outer scopes
		if e.outer != nil {
			return e.outer.GetRef(name)
		}
		return nil, false
	}
	// Ensure refStore exists
	if e.refStore == nil {
		e.refStore = make(map[string]*EnvEntry)
	}
	// Get or create entry
	ref, ok := e.refStore[name]
	if !ok {
		ref = &EnvEntry{Owner: e, Name: name, Value: e.store[name]}
		e.refStore[name] = ref
	}
	return ref, true
}

// SetByPointer sets a value through a pointer reference.
func (e *Environment) SetByPointer(ref *EnvEntry, val Object) {
	if ref == nil || ref.Owner == nil || ref.Name == "" {
		return
	}

	// Update the reference
	ref.Value = val

	owner := ref.Owner
	owner.store[ref.Name] = val
	// Sync slot if this variable has one
	owner.SetSlotByName(ref.Name, val)
	if shouldTrackType(string(val.Type())) {
		owner.ensureTypes()
		owner.types[ref.Name] = string(val.Type())
	}
}

// allocInteger returns an Integer with the given value.
// For cached range: returns pointer into the global cache (zero alloc).
// For out-of-range: reuses a recycled Integer from the freelist if available,
// otherwise heap-allocates a new one.
func (e *Environment) allocInteger(value int64) *Integer {
	if value >= integerCacheMin && value <= integerCacheMax {
		return &integerCache[value-integerCacheMin]
	}
	e.freelistMu.Lock()
	if n := len(e.intFreelist); n > 0 {
		obj := e.intFreelist[n-1]
		e.intFreelist[n-1] = nil // avoid retaining reference
		e.intFreelist = e.intFreelist[:n-1]
		e.freelistMu.Unlock()
		obj.Value = value
		return obj
	}
	e.freelistMu.Unlock()
	return &Integer{Value: value}
}

// recycleInteger adds an Integer object back to the freelist for reuse.
// Cached Integers (pointers into the global integerCache) are not recycled.
func (e *Environment) recycleInteger(obj *Integer) {
	if isCachedInteger(obj) {
		return
	}
	e.freelistMu.Lock()
	if len(e.intFreelist) < intFreelistCap {
		e.intFreelist = append(e.intFreelist, obj)
	}
	e.freelistMu.Unlock()
}

// isCachedInteger reports whether obj is a pointer into the global integerCache.
func isCachedInteger(obj *Integer) bool {
	return uintptr(unsafe.Pointer(obj)) >= uintptr(unsafe.Pointer(&integerCache[0])) &&
		uintptr(unsafe.Pointer(obj)) < uintptr(unsafe.Pointer(&integerCache[0]))+uintptr(len(integerCache))*unsafe.Sizeof(integerCache[0])
}

// GetBySlot retrieves a value by scope depth and slot index.
// depth=0 means current scope, depth=1 means one level up, etc.
func (e *Environment) GetBySlot(depth, index int) Object {
	scope := e
	for i := 0; i < depth; i++ {
		if scope.outer == nil {
			return nil
		}
		scope = scope.outer
	}
	if scope.slots == nil || index < 0 || index >= len(scope.slots) {
		return nil
	}
	return scope.slots[index]
}

// SetSlot sets a value by scope depth and slot index.
func (e *Environment) SetSlot(depth, index int, val Object) {
	scope := e
	for i := 0; i < depth; i++ {
		if scope.outer == nil {
			return
		}
		scope = scope.outer
	}
	if scope.slots == nil || index < 0 || index >= len(scope.slots) {
		return
	}
	scope.slots[index] = val
}

// DeclareSlot allocates a slot in the current scope and stores the value.
// Also stores in the map for backward compatibility (pointer ops, Keys(), etc.).
// If the variable already has a slot, updates the existing slot instead of allocating a new one.
// Returns the allocated slot index.
func (e *Environment) DeclareSlot(name string, val Object, typeName string) int {
	if e.slots == nil {
		e.slots = make([]Object, 0, 4)
	}
	if e.slotByName == nil {
		e.slotByName = make(map[string]int)
	}
	// Check if this variable already has a slot (e.g., re-declaration in a loop body)
	if idx, ok := e.slotByName[name]; ok {
		e.slots[idx] = val
		// Also update map
		e.store[name] = val
		if shouldTrackType(typeName) {
			e.ensureTypes()
			e.types[name] = typeName
		}
		return idx
	}
	idx := len(e.slots)
	e.slots = append(e.slots, val)
	e.slotByName[name] = idx

	// Also store in map for backward compatibility
	e.store[name] = val
	if shouldTrackType(typeName) {
		e.ensureTypes()
		e.types[name] = typeName
	}
	return idx
}

// SetSlotByName sets a value in the slot associated with name (if any).
// Returns true if a slot was found and updated.
func (e *Environment) SetSlotByName(name string, val Object) bool {
	if e.slotByName == nil {
		return false
	}
	idx, ok := e.slotByName[name]
	if !ok {
		return false
	}
	if e.slots != nil && idx < len(e.slots) {
		e.slots[idx] = val
		return true
	}
	return false
}
