package core

type TokenType string

const (
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	// Identifiers + literals
	IDENT  TokenType = "IDENT"
	STRING TokenType = "STRING"
	NUMBER TokenType = "NUMBER"
	FLOAT  TokenType = "FLOAT"

	// Operators
	ASSIGN TokenType = "="
	COLON_ASSIGN TokenType = ":="
	PIPE TokenType = "|"
	GREATER TokenType = ">"
	LESS TokenType = "<"
	APPEND TokenType = ">>"
	REDIRECT TokenType = "->"
	AND TokenType = "&&"
	AMPERSAND TokenType = "&"
	OR TokenType = "||"
	NOT TokenType = "!"
	EQ TokenType = "=="
	NEQ TokenType = "!="
	PLUS TokenType = "+"
	MINUS TokenType = "-"
	ASTERISK TokenType = "*"
	SEMICOLON TokenType = ";"
	COMMA TokenType = ","
	DOT TokenType = "."
	LPAREN TokenType = "("
	RPAREN TokenType = ")"
	LBRACE TokenType = "{"
	RBRACE TokenType = "}"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"
	DOLLAR TokenType = "$"

	// Keywords
	IF        TokenType = "IF"
	ELSE      TokenType = "ELSE"
	FOR       TokenType = "FOR"
	RANGE     TokenType = "RANGE"
	FUNC      TokenType = "FUNC"
	RETURN    TokenType = "RETURN"
	GO        TokenType = "GO"
	VAR       TokenType = "VAR"
	PRINT     TokenType = "PRINT"
	EXEC      TokenType = "EXEC"
	NIL       TokenType = "NIL"
	TRUE_TOK  TokenType = "TRUE"
	FALSE_TOK TokenType = "FALSE"
	IMPORT    TokenType = "IMPORT"
	WAIT      TokenType = "WAIT"
	SWITCH    TokenType = "SWITCH"
	CASE      TokenType = "CASE"
	DEFAULT   TokenType = "DEFAULT"
	BREAK     TokenType = "BREAK"
	CONTINUE  TokenType = "CONTINUE"
	COLON     TokenType = ":"
)

type Token struct {
	Type    TokenType
	Literal string
	Start   int
	End     int
}

func LookupIdent(ident string) TokenType {
	switch ident {
	case "if":
		return IF
	case "else":
		return ELSE
	case "for":
		return FOR
	case "range":
		return RANGE
	case "func":
		return FUNC
	case "return":
		return RETURN
	case "go":
		return GO
	case "var":
		return VAR
	case "print":
		return PRINT
	case "exec":
		return EXEC
	case "nil":
		return NIL
	case "true":
		return TRUE_TOK
	case "false":
		return FALSE_TOK
	case "import":
		return IMPORT
	case "wait":
		return WAIT
	case "switch":
		return SWITCH
	case "case":
		return CASE
	case "default":
		return DEFAULT
	case "break":
		return BREAK
	case "continue":
		return CONTINUE
	default:
		return IDENT
	}
}
