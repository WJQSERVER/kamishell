package core

import (
	"io"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkEnvironmentSet(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		env.Set("key", "value")
	}
}

func BenchmarkEnvironmentGetNested(b *testing.B) {
	root := NewEmptyEnvironment()
	root.Set("answer", int64(42))

	env := root
	for range 8 {
		env = NewEnclosedEnvironment(env)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value, ok := env.Get("answer")
		if !ok || value.(*Integer).Value != 42 {
			b.Fatalf("unexpected lookup result: %v, %v", value, ok)
		}
	}
}

func BenchmarkPackageSnapshot(b *testing.B) {
	env := NewScriptEnvironment(NewEmptyEnvironment())
	for i := range 64 {
		env.SetPackageValue("env", "KEY_"+strconv.Itoa(i), strconv.Itoa(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot := env.PackageSnapshot("env")
		if len(snapshot) != 64 {
			b.Fatalf("unexpected snapshot size: %d", len(snapshot))
		}
	}
}

func BenchmarkEnvironmentClone(b *testing.B) {
	env := NewScriptEnvironment(NewEmptyEnvironment())
	for i := range 128 {
		key := "key_" + strconv.Itoa(i)
		env.Set(key, int64(i))
		env.SetPackageValue("env", "KEY_"+strconv.Itoa(i), strconv.Itoa(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := env.Clone()
		if value, ok := clone.GetObject("key_64"); !ok || value.(*Integer).Value != 64 {
			b.Fatalf("unexpected clone result: %v, %v", value, ok)
		}
	}
}

func BenchmarkEnvironmentAssignNested(b *testing.B) {
	root := NewEmptyEnvironment()
	root.SetWithType("answer", getIntegerObject(41), string(INTEGER_OBJ))

	env := root
	for range 8 {
		env = NewEnclosedEnvironment(env)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.Assign("answer", getIntegerObject(42))
		value, ok := root.GetObject("answer")
		if !ok || value.(*Integer).Value != 42 {
			b.Fatalf("unexpected assign result: %v, %v", value, ok)
		}
		root.Assign("answer", getIntegerObject(41))
	}
}

func BenchmarkEvalAssignmentNested(b *testing.B) {
	program := mustParseBenchmarkProgram(b, `answer = 42`)
	root := NewEmptyEnvironment()
	root.SetWithType("answer", getIntegerObject(41), string(INTEGER_OBJ))

	env := root
	for range 8 {
		env = NewEnclosedEnvironment(env)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
		value, ok := root.GetObject("answer")
		if !ok || value.(*Integer).Value != 42 {
			b.Fatalf("unexpected assignment result: %v, %v", value, ok)
		}
		root.Assign("answer", getIntegerObject(41))
	}
}
