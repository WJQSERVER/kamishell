package core

import (
	"strconv"
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
	for i := 0; i < 8; i++ {
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
	for i := 0; i < 64; i++ {
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
