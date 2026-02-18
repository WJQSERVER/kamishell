package kamishell

import (
	"testing"
)

func BenchmarkNextToken(b *testing.B) {
	input := `print "hello";
	files := ls -la;
	if err != nil {
		exit 1
	}`

	for i := 0; i < b.N; i++ {
		l := NewLexer(input)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}
