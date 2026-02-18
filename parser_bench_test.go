package kamishell

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

	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()
	}
}
