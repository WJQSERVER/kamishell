package recompiler

// Ptr represents a pointer to a variable in compiled Kami code.
// Uses closures to capture variable references without relying on Env.
type Ptr struct {
	Get func() any
	Set func(any)
}

// NewPtr creates a pointer to a variable.
// get returns the current value, set updates it.
func NewPtr(get func() any, set func(any)) *Ptr {
	return &Ptr{Get: get, Set: set}
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
