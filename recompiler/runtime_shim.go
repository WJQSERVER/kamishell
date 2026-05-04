package recompiler

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"kamishell/builtin"
	"kamishell/kamilib"

	"github.com/valyala/bytebufferpool"
)

type Env struct {
	store map[string]string // primary store: strings only, no any boxing
	mu    sync.RWMutex
}

func NewEnv() *Env {
	e := &Env{store: make(map[string]string)}
	for _, env := range os.Environ() {
		k, v, _ := strings.Cut(env, "=")
		e.store[k] = v
	}
	return e
}

// SetString stores a string value directly (no boxing).
func (e *Env) SetString(name string, val string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = val
}

// GetString retrieves a string value directly (no boxing).
func (e *Env) GetString(name string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	v, ok := e.store[name]
	return v, ok
}

// Set implements builtin.Environment — converts any to string via ToStr.
func (e *Env) Set(name string, val any) {
	e.SetString(name, ToStr(val))
}

// Get implements builtin.Environment — returns string as any.
func (e *Env) Get(name string) (any, bool) {
	return e.GetString(name)
}

func (e *Env) SetInt(name string, v int64)    { e.SetString(name, strconv.FormatInt(v, 10)) }
func (e *Env) SetFloat(name string, v float64) { e.SetString(name, strconv.FormatFloat(v, 'f', -1, 64)) }
func (e *Env) SetStr(name string, v string)    { e.SetString(name, v) }
func (e *Env) SetBool(name string, v bool)     { e.SetString(name, strconv.FormatBool(v)) }

func (e *Env) GetInt(name string) int64 {
	v, ok := e.GetString(name)
	if !ok {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func (e *Env) GetStr(name string) string {
	v, _ := e.GetString(name)
	return v
}

// Task represents a goroutine future result.
type Task struct {
	done chan struct{}
	res  any
	err  error
}

func NewTask() *Task {
	return &Task{done: make(chan struct{})}
}

func (t *Task) SetResult(v any) {
	t.res = v
	close(t.done)
}

func (t *Task) SetError(e error) {
	t.err = e
	close(t.done)
}

func (t *Task) Wait() (any, error) {
	<-t.done
	return t.res, t.err
}

func (t *Task) WaitTimeout(secs int64) (any, error) {
	select {
	case <-t.done:
		return t.res, t.err
	case <-time.After(time.Duration(secs) * time.Second):
		return nil, fmt.Errorf("timeout")
	}
}

var (
	jobsMu    sync.Mutex
	jobs      = make(map[int]*builtin.Job)
	nextJobID = 1
)

func registerJob(cmd string) int {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	id := nextJobID
	jobs[id] = &builtin.Job{ID: id, Command: cmd, Status: "Running"}
	nextJobID++
	return id
}

func completeJob(id int, success bool, errMsg string) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	if j, ok := jobs[id]; ok {
		if success {
			j.Status = "Done"
		} else {
			j.Status = "Failed"
			j.Error = errMsg
		}
	}
}

func allJobsDone() bool {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	for _, j := range jobs {
		if j.Status == "Running" {
			return false
		}
	}
	return true
}

// waitAll blocks until all registered background jobs complete.
func waitAll() {
	for !allJobsDone() {
		time.Sleep(10 * time.Millisecond)
	}
}

func waitAllTimeout(secs int64) {
	deadline := time.After(time.Duration(secs) * time.Second)
	for !allJobsDone() {
		select {
		case <-deadline:
			return
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// String interpolation: $var expansion with env fallback.
func ExpandStr(s string, env *Env) string {
	return os.Expand(s, func(name string) string {
		if v, ok := env.Get(name); ok {
			return ToStr(v)
		}
		return os.Getenv(name)
	})
}

// Truthy check: Kamishell truthiness semantics.
func IsTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	}
	return true
}

// Type-dispatch numeric operations.
func Add(a, b any) any {
	ai, aIsInt := a.(int64)
	bi, bIsInt := b.(int64)
	if aIsInt && bIsInt {
		return ai + bi
	}
	bb := bytebufferpool.Get()
	bb.B = kamilib.AppendAny(bb.B, a)
	bb.B = kamilib.AppendAny(bb.B, b)
	s := string(bb.B)
	bytebufferpool.Put(bb)
	return s
}

func Sub(a, b any) any {
	ai, aIsInt := a.(int64)
	bi, bIsInt := b.(int64)
	if aIsInt && bIsInt {
		return ai - bi
	}
	return int64(0)
}

func Mul(a, b any) any {
	ai, aIsInt := a.(int64)
	bi, bIsInt := b.(int64)
	if aIsInt && bIsInt {
		return ai * bi
	}
	return int64(0)
}

func Div(a, b any) any {
	ai, aIsInt := a.(int64)
	bi, bIsInt := b.(int64)
	if aIsInt && bIsInt && bi != 0 {
		return ai / bi
	}
	return int64(0)
}

func Eq(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch x := a.(type) {
	case int64:
		if y, ok := b.(int64); ok {
			return x == y
		}
	case float64:
		if y, ok := b.(float64); ok {
			return x == y
		}
	case string:
		if y, ok := b.(string); ok {
			return x == y
		}
	case bool:
		if y, ok := b.(bool); ok {
			return x == y
		}
	}
	return ToStr(a) == ToStr(b)
}

func NotEq(a, b any) bool { return !Eq(a, b) }

func LessThan(a, b any) bool {
	switch x := a.(type) {
	case int64:
		if y, ok := b.(int64); ok {
			return x < y
		}
	case float64:
		if y, ok := b.(float64); ok {
			return x < y
		}
	}
	return false
}

func GreaterThan(a, b any) bool {
	switch x := a.(type) {
	case int64:
		if y, ok := b.(int64); ok {
			return x > y
		}
	case float64:
		if y, ok := b.(float64); ok {
			return x > y
		}
	}
	return false
}

func LessEq(a, b any) bool {
	switch x := a.(type) {
	case int64:
		if y, ok := b.(int64); ok {
			return x <= y
		}
	case float64:
		if y, ok := b.(float64); ok {
			return x <= y
		}
	}
	return false
}

func GreaterEq(a, b any) bool {
	switch x := a.(type) {
	case int64:
		if y, ok := b.(int64); ok {
			return x >= y
		}
	case float64:
		if y, ok := b.(float64); ok {
			return x >= y
		}
	}
	return false
}

// ToStr converts any value to its string representation.
func ToStr(v any) string {
	if v == nil {
		return "nil"
	}
	switch x := v.(type) {
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return x.String()
	}
	bb := bytebufferpool.Get()
	bb.B = kamilib.AppendAny(bb.B, v)
	s := string(bb.B)
	bytebufferpool.Put(bb)
	return s
}

// Array helpers.
func ArrayLen(arr any) int64 {
	switch x := arr.(type) {
	case []any:
		return int64(len(x))
	case []int64:
		return int64(len(x))
	case []string:
		return int64(len(x))
	case []float64:
		return int64(len(x))
	case []bool:
		return int64(len(x))
	case string:
		return int64(len(x))
	}
	return 0
}

func ArrayGet(arr any, idx int64) any {
	switch x := arr.(type) {
	case []any:
		if idx >= 0 && idx < int64(len(x)) {
			return x[idx]
		}
	case []int64:
		if idx >= 0 && idx < int64(len(x)) {
			return x[idx]
		}
	case []string:
		if idx >= 0 && idx < int64(len(x)) {
			return x[idx]
		}
	case []float64:
		if idx >= 0 && idx < int64(len(x)) {
			return x[idx]
		}
	case []bool:
		if idx >= 0 && idx < int64(len(x)) {
			return x[idx]
		}
	}
	return nil
}

func ArraySet(arr any, idx int64, val any) any {
	switch x := arr.(type) {
	case []any:
		if idx >= 0 && idx < int64(len(x)) {
			x[idx] = val
			return x
		}
	case []int64:
		if idx >= 0 && idx < int64(len(x)) {
			if v, ok := val.(int64); ok {
				x[idx] = v
				return x
			}
		}
	case []string:
		if idx >= 0 && idx < int64(len(x)) {
			if v, ok := val.(string); ok {
				x[idx] = v
				return x
			}
		}
	case []float64:
		if idx >= 0 && idx < int64(len(x)) {
			if v, ok := val.(float64); ok {
				x[idx] = v
				return x
			}
		}
	case []bool:
		if idx >= 0 && idx < int64(len(x)) {
			if v, ok := val.(bool); ok {
				x[idx] = v
				return x
			}
		}
	}
	return arr
}

func ArrayPush(arr any, val any) any {
	switch x := arr.(type) {
	case []any:
		return append(x, val)
	case []int64:
		if v, ok := val.(int64); ok {
			return append(x, v)
		}
	case []string:
		if v, ok := val.(string); ok {
			return append(x, v)
		}
	case []float64:
		if v, ok := val.(float64); ok {
			return append(x, v)
		}
	case []bool:
		if v, ok := val.(bool); ok {
			return append(x, v)
		}
	}
	return arr
}

// PrependImport registers a Go import path for the generated file.
// Not a runtime operation; used by the compiler to track imports.
var GeneratedImports []string

func ResetImports() { GeneratedImports = nil }

// RegisterGoJob registers a goroutine-based background job and returns its ID.
func RegisterGoJob(cmd string) int {
	return registerJob(cmd)
}

// CompleteGoJob marks a goroutine job as done.
func CompleteGoJob(id int) {
	completeJob(id, true, "")
}

// WaitAll blocks until all background jobs finish.
func WaitAll() {
	waitAll()
}

// WaitAllTimeout blocks until all jobs finish or timeout.
func WaitAllTimeout(secs any) {
	switch v := secs.(type) {
	case int64:
		waitAllTimeout(v)
	case float64:
		waitAllTimeout(int64(v))
	default:
		waitAll()
	}
}

// MemberGet gets a member value from an object.
func MemberGet(obj any, prop string) any {
	return nil
}

// NewError creates a new error value (for the error() native function).
type kamiError struct {
	message string
}

func (e *kamiError) Error() string { return e.message }

func NewError(msg string) error {
	return &kamiError{message: msg}
}

// CallFunc calls a function value with the given arguments.
func CallFunc(fn any, env *Env, args ...any) any {
	switch f := fn.(type) {
	case func(*Env, ...any) any:
		return f(env, args...)
	case func(...any) any:
		return f(args...)
	case func(any, any) any:
		if len(args) >= 2 {
			return f(args[0], args[1])
		}
	case func(any) any:
		if len(args) >= 1 {
			return f(args[0])
		}
	case func() any:
		return f()
	}
	return nil
}