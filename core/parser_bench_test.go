package core

import (
	"testing"
)

func BenchmarkParseProgram(b *testing.B) {
	input := `x := 5;
	y := true;
	if x {
		print "yes";
	} else {
		ls -la;
	}`
	benchmarkParseProgram(b, input)
}

func BenchmarkParseProgramControlFlow(b *testing.B) {
	input := `func greet(name, times) {
	i := 0
	for i < times {
		print "hello $name"
		i = i + 1
	}
}

user := "kami"
if user == "kami" {
	greet(user, 3)
} else {
	print "unknown"
}`
	benchmarkParseProgram(b, input)
}

func BenchmarkParseProgramEnvAndCalls(b *testing.B) {
	input := `env.Set("GOOS", "linux")
env.Set("GOARCH", "amd64")
target_env "app" "CGO_ENABLED=0"
print env.Get("GOOS")`
	benchmarkParseProgram(b, input)
}

func benchmarkParseProgram(b *testing.B, input string) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()
	}
}
