package recompiler

// Ptr represents a pointer to a variable in compiled Kami code.
// Uses closures to capture variable references without relying on Env.
type Ptr struct {
	Get func() any
	Set func(any)
}

// NewPtr creates a pointer to a variable with any-typed closures.
// get returns the current value, set updates it.
func NewPtr(get func() any, set func(any)) *Ptr {
	return &Ptr{Get: get, Set: set}
}

// NewPtrInt64 creates a pointer to an int64 variable.
func NewPtrInt64(get func() int64, set func(int64)) *Ptr {
	return &Ptr{
		Get: func() any { return get() },
		Set: func(v any) { set(v.(int64)) },
	}
}

// NewPtrString creates a pointer to a string variable.
func NewPtrString(get func() string, set func(string)) *Ptr {
	return &Ptr{
		Get: func() any { return get() },
		Set: func(v any) { set(v.(string)) },
	}
}

// NewPtrFloat64 creates a pointer to a float64 variable.
func NewPtrFloat64(get func() float64, set func(float64)) *Ptr {
	return &Ptr{
		Get: func() any { return get() },
		Set: func(v any) { set(v.(float64)) },
	}
}

// NewPtrBool creates a pointer to a bool variable.
func NewPtrBool(get func() bool, set func(bool)) *Ptr {
	return &Ptr{
		Get: func() any { return get() },
		Set: func(v any) { set(v.(bool)) },
	}
}

// Deref dereferences a pointer to get its value.
// Accepts *Ptr or any (will try type assertion).
func Deref(p any) any {
	ptr, ok := p.(*Ptr)
	if !ok || ptr == nil || ptr.Get == nil {
		return nil
	}
	return ptr.Get()
}

// SetPtr sets the value through a pointer.
// Accepts *Ptr or any (will try type assertion).
func SetPtr(p any, val any) {
	ptr, ok := p.(*Ptr)
	if !ok || ptr == nil || ptr.Set == nil {
		return
	}
	ptr.Set(val)
}
