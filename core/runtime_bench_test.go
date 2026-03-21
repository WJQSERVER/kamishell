package core

import (
	"io"
	"strings"
	"testing"
)

func BenchmarkEvalArithmeticProgram(b *testing.B) {
	benchmarkEvalProgram(b, `x := 10 + 20; y := x + 30; print (y + 40)`)
}

func BenchmarkEvalLoopProgram(b *testing.B) {
	benchmarkEvalProgram(b, `i := 0; for i < 100 { i = i + 1 }; print i`)
}

func BenchmarkEvalFunctionCallProgram(b *testing.B) {
	benchmarkEvalProgram(b, `func greet(name) { print name }; greet("kami")`)
}

func BenchmarkEvalEnvPackageProgram(b *testing.B) {
	benchmarkEvalProgram(b, `env.Set("GOOS", "linux"); env.Get("GOOS")`)
}

func BenchmarkEvalPipelineProgram(b *testing.B) {
	benchmarkEvalProgram(b, "print \"line1\\nline2\" | cat")
}

func BenchmarkExecuteCommandBuiltinCat(b *testing.B) {
	env := NewEmptyEnvironment()
	payload := strings.Repeat("hello world\n", 32)

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := executeCommand("cat", nil, env, strings.NewReader(payload), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func benchmarkEvalProgram(b *testing.B, input string) {
	b.Helper()
	program := mustParseBenchmarkProgram(b, input)

	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func mustParseBenchmarkProgram(b *testing.B, input string) *Program {
	b.Helper()
	l := NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	if program == nil {
		b.Fatal("expected parsed program")
	}
	return program
}
