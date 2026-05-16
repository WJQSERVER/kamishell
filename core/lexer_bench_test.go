package core

import (
	"testing"
)

func BenchmarkNextToken(b *testing.B) {
	input := `print "hello";
	files := ls "-la";
	if err != nil {
		exit 1
	}`
	benchmarkNextToken(b, input)
}

func BenchmarkNextTokenLargeScript(b *testing.B) {
	input := `project "bench"
env.Set("GOOS", "linux")
env.Set("GOARCH", "amd64")
func greet(name, count) {
	i := 0
	for i < count {
		print "hello $name"
		i = i + 1
	}
}
greet("kami", 5)
print "done" | cat`
	benchmarkNextToken(b, input)
}

func BenchmarkNextTokenStringsAndPaths(b *testing.B) {
	input := `path := "C:\\tools\\kami"
config := "${HOME}/.kami/config"
print "Path: $path, Config: $config"
exec "go build -o bin/kami main.go"`
	benchmarkNextToken(b, input)
}

func benchmarkNextToken(b *testing.B, input string) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))

	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}
