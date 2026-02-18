package lexer

type TokenType string

const (
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	// Identifiers + literals
	IDENT  TokenType = "IDENT"
	STRING TokenType = "STRING"
	NUMBER TokenType = "NUMBER"

	// Operators
	ASSIGN  TokenType = "="
	COLON_ASSIGN TokenType = ":="
	PIPE    TokenType = "|"
	GREATER TokenType = ">"
	APPEND  TokenType = ">>"
	AND     TokenType = "&&"
	OR      TokenType = "||"
	NOT     TokenType = "!"
	EQ      TokenType = "=="
	NEQ     TokenType = "!="
	SEMICOLON TokenType = ";"
	COMMA     TokenType = ","
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"

	// Keywords
	IF     TokenType = "IF"
	ELSE   TokenType = "ELSE"
	FOR    TokenType = "FOR"
	RANGE  TokenType = "RANGE"
	FUNC   TokenType = "FUNC"
	RETURN TokenType = "RETURN"
	GO     TokenType = "GO"
	VAR    TokenType = "VAR"
	PRINT  TokenType = "PRINT"
	NIL    TokenType = "NIL"
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"if":     IF,
	"else":   ELSE,
	"for":    FOR,
	"range":  RANGE,
	"func":   FUNC,
	"return": RETURN,
	"go":     GO,
	"var":    VAR,
	"print":  PRINT,
	"nil":    NIL,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
