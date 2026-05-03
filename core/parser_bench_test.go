package core

import (
	"strings"
	"testing"
)

func BenchmarkParseProgram(b *testing.B) {
	input := `x := 5;
	y := true;
	if x {
		print "yes";
	} else {
		ls "-la";
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

func BenchmarkParseProgramCommandHeavy(b *testing.B) {
	input := `target_env "app" GOOS=linux GOARCH=amd64 CGO_ENABLED=0
http --method POST --header "Content-Type: application/json" --data '{"name":"kami"}' "https://api.example.com/items"
grep -n main go.mod | cat && print "done" &`
	benchmarkParseProgram(b, input)
}

func BenchmarkParseProgramScaling(b *testing.B) {
	inputs := map[string]string{
		"small":  repeatBenchmarkSnippet(`print "hello $name"; target_env "app" GOOS=linux GOARCH=amd64`, 1),
		"medium": repeatBenchmarkSnippet(`print "hello $name"; target_env "app" GOOS=linux GOARCH=amd64`, 25),
		"large":  repeatBenchmarkSnippet(`print "hello $name"; target_env "app" GOOS=linux GOARCH=amd64`, 100),
	}

	for name, input := range inputs {
		b.Run(name, func(b *testing.B) {
			benchmarkParseProgram(b, input)
		})
	}
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

func repeatBenchmarkSnippet(snippet string, times int) string {
	if times <= 1 {
		return snippet
	}

	var result strings.Builder
	for i := range times {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(snippet)
	}
	return result.String()
}
