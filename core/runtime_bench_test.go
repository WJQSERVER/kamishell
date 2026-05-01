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

func BenchmarkEvalLiteralHeavyProgram(b *testing.B) {
	benchmarkEvalProgram(b, `i := 0; for i < 100 { print "tick"; i = i + 1 }`)
}

func BenchmarkEvalPipelineProgram(b *testing.B) {
	benchmarkEvalProgram(b, "print \"line1\\nline2\" | cat")
}

func BenchmarkEvalInterpolatedStringProgram(b *testing.B) {
	benchmarkEvalProgram(b, `name := "kami"; print "hello $name from $name"`)
}

func BenchmarkExecuteCommandUserFunction(b *testing.B) {
	env := NewEmptyEnvironment()
	fn := &Function{
		Parameters: []string{"value"},
		Body: &BlockStatement{Statements: []Statement{
			&ExpressionStatement{Expression: &Identifier{Value: "value"}},
		}},
		Env: env,
	}
	env.Set("identity", fn)
	args := []Expression{&IntegerLiteral{Value: 7, Obj: getIntegerObject(7)}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := executeCommand("identity", args, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
		str, ok := result.(*String)
		if !ok || str.Value != "7" {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
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

func BenchmarkEvalPipelineScaling(b *testing.B) {
	benchmarks := map[string]string{
		"two-stages":   `print "line1\nline2" | cat`,
		"three-stages": `print "line1\nline2" | cat | cat`,
		"four-stages":  `print "line1\nline2" | cat | cat | cat`,
	}

	for name, input := range benchmarks {
		b.Run(name, func(b *testing.B) {
			benchmarkEvalProgram(b, input)
		})
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

// Pointer operation benchmarks

func BenchmarkPointerAddressOf(b *testing.B) {
	// Benchmark &x (address-of)
	program := mustParseBenchmarkProgram(b, `x := 42; p := &x; print *p`)
	b.ReportAllocs()
	b.SetBytes(int64(len(`x := 42; p := &x; print *p`)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func BenchmarkPointerDereference(b *testing.B) {
	// Benchmark *p (dereference) - read only
	program := mustParseBenchmarkProgram(b, `x := 42; p := &x; val := *p; print val`)
	b.ReportAllocs()
	b.SetBytes(int64(len(`x := 42; p := &x; val := *p; print val`)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func BenchmarkPointerAssign(b *testing.B) {
	// Benchmark *p = val (pointer assignment)
	program := mustParseBenchmarkProgram(b, `x := 42; p := &x; *p = 100; print x`)
	b.ReportAllocs()
	b.SetBytes(int64(len(`x := 42; p := &x; *p = 100; print x`)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func BenchmarkPointerPassToFunction(b *testing.B) {
	// Benchmark passing pointer to function
	program := mustParseBenchmarkProgram(b, `func inc(p) { *p = *p + 1 }; x := 0; p := &x; inc(p); print x`)
	b.ReportAllocs()
	b.SetBytes(int64(len(`func inc(p) { *p = *p + 1 }; x := 0; p := &x; inc(p); print x`)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := NewEmptyEnvironment()
		result := EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		if isError(result) {
			b.Fatalf("unexpected error: %s", result.Inspect())
		}
	}
}

func BenchmarkPointerVsDirectAssign(b *testing.B) {
	// Compare: direct assignment vs pointer assignment
	b.Run("DirectAssign", func(b *testing.B) {
		program := mustParseBenchmarkProgram(b, `x := 0; x = 100; print x`)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			env := NewEmptyEnvironment()
			EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		}
	})
	b.Run("PointerAssign", func(b *testing.B) {
		program := mustParseBenchmarkProgram(b, `x := 0; p := &x; *p = 100; print x`)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			env := NewEmptyEnvironment()
			EvalWithIO(program, env, strings.NewReader(""), io.Discard, io.Discard)
		}
	})
}
