package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	prevToken    TokenType
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	l.skipShebang()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *Lexer) skipShebang() {
	if l.ch == '#' && l.peekChar() == '!' {
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readChar()
		}
	}
}

func (l *Lexer) NextToken() Token {
	var tok Token

	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
		} else if l.ch == '\n' {
			if l.isCompletable() {
				l.readChar()
				tok = Token{Type: SEMICOLON, Literal: ";"}
				tok.Start = l.position
				tok.End = l.position + 1
				l.prevToken = SEMICOLON
				return tok
			}
			l.readChar()
		} else if l.ch == '/' {
			if l.peekChar() == '/' {
				l.skipSingleLineComment()
			} else if l.peekChar() == '*' {
				l.skipMultiLineComment()
			} else {
				break
			}
		} else {
			break
		}
	}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			start := l.position
			l.readChar()
			tok = Token{Type: EQ, Literal: "==", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: ASSIGN, Literal: "=", Start: l.position, End: l.position + 1}
		}
	case ':':
		if l.peekChar() == '=' {
			start := l.position
			l.readChar()
			tok = Token{Type: COLON_ASSIGN, Literal: ":=", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Start: l.position, End: l.position + 1}
		}
	case '|':
		if l.peekChar() == '|' {
			start := l.position
			l.readChar()
			tok = Token{Type: OR, Literal: "||", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: PIPE, Literal: "|", Start: l.position, End: l.position + 1}
		}
	case '&':
		if l.peekChar() == '&' {
			start := l.position
			l.readChar()
			tok = Token{Type: AND, Literal: "&&", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: AMPERSAND, Literal: "&", Start: l.position, End: l.position + 1}
		}
	case '>':
		if l.peekChar() == '>' {
			start := l.position
			l.readChar()
			tok = Token{Type: APPEND, Literal: ">>", Start: start, End: l.readPosition}
		} else if l.peekChar() == '-' {
			start := l.position
			l.readChar()
			tok = Token{Type: REDIRECT, Literal: "->", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: GREATER, Literal: ">", Start: l.position, End: l.position + 1}
		}
	case '<':
		tok = Token{Type: LESS, Literal: "<", Start: l.position, End: l.position + 1}
	case '!':
		if l.peekChar() == '=' {
			start := l.position
			l.readChar()
			tok = Token{Type: NEQ, Literal: "!=", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: NOT, Literal: "!", Start: l.position, End: l.position + 1}
		}
	case '+':
		tok = Token{Type: PLUS, Literal: "+", Start: l.position, End: l.position + 1}
	case '-':
		if l.peekChar() == '>' {
			start := l.position
			l.readChar()
			tok = Token{Type: REDIRECT, Literal: "->", Start: start, End: l.readPosition}
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Start: l.position, End: l.position + 1}
		}
	case ';':
		tok = Token{Type: SEMICOLON, Literal: ";", Start: l.position, End: l.position + 1}
	case ',':
		tok = Token{Type: COMMA, Literal: ",", Start: l.position, End: l.position + 1}
	case '.':
		if isDigit(l.peekChar()) {
			start := l.position
			tok.Type = FLOAT
			tok.Literal = "0" + l.readFloat()
			tok.Start = start
			tok.End = l.position
			l.prevToken = tok.Type
			return tok
		}
		tok = Token{Type: DOT, Literal: ".", Start: l.position, End: l.position + 1}
	case '(':
		tok = Token{Type: LPAREN, Literal: "(", Start: l.position, End: l.position + 1}
	case ')':
		tok = Token{Type: RPAREN, Literal: ")", Start: l.position, End: l.position + 1}
	case '{':
		tok = Token{Type: LBRACE, Literal: "{", Start: l.position, End: l.position + 1}
	case '}':
		tok = Token{Type: RBRACE, Literal: "}", Start: l.position, End: l.position + 1}
	case '$':
		tok = Token{Type: DOLLAR, Literal: "$", Start: l.position, End: l.position + 1}
	case '"':
		start := l.position
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Start = start
		tok.End = l.position + 1
	case 0:
		if l.isCompletable() {
			tok = Token{Type: SEMICOLON, Literal: ";", Start: l.position, End: l.position}
			l.prevToken = SEMICOLON
			return tok
		}
		tok.Literal = ""
		tok.Type = EOF
		tok.Start = l.position
		tok.End = l.position
	default:
		if isIdentifierStart(l.input[l.position:]) {
			start := l.position
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			tok.Start = start
			tok.End = l.position
			l.prevToken = tok.Type
			return tok
		} else if isDigit(l.ch) {
			start := l.position
			literal, tokenType := l.readNumberOrFloat()
			tok.Type = tokenType
			tok.Literal = literal
			tok.Start = start
			tok.End = l.position
			l.prevToken = tok.Type
			return tok
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Start: l.position, End: l.position + 1}
		}
	}

	l.readChar()
	l.prevToken = tok.Type
	return tok
}

func (l *Lexer) isCompletable() bool {
	switch l.prevToken {
	case IDENT, NUMBER, FLOAT, STRING, TRUE_TOK, FALSE_TOK, NIL, RETURN, RPAREN, RBRACE:
		return true
	}
	return false
}

func (l *Lexer) skipSingleLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipMultiLineComment() {
	l.readChar() // consume '/'
	l.readChar() // consume '*'
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume '*'
			l.readChar() // consume '/'
			break
		}
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for {
		if l.ch == 0 {
			break
		}
		if l.ch < utf8.RuneSelf {
			if !isASCIIIdentifierPart(l.ch) {
				break
			}
			l.advanceBytes(1)
		} else {
			_, size := utf8.DecodeRuneInString(l.input[l.position:])
			if size <= 0 {
				break
			}
			l.advanceBytes(size)
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readFloat() string {
	position := l.position
	if l.ch == '.' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumberOrFloat() (string, TokenType) {
	start := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
		return l.input[start:l.position], FLOAT
	}
	return l.input[start:l.position], NUMBER
}

func (l *Lexer) readString() string {
	position := l.position + 1
	var out strings.Builder
	hasEscape := false
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
			hasEscape = true
			if out.Cap() == 0 {
				out.Grow(l.position - position)
				out.WriteString(l.input[position:l.position])
			}
			l.readChar()
			switch l.ch {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			default:
				out.WriteByte('\\')
				out.WriteByte(l.ch)
			}
		} else if hasEscape {
			out.WriteByte(l.ch)
		}
	}
	if hasEscape {
		return out.String()
	}
	return l.input[position:l.position]
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func isLetter(ch byte) bool {
	return isASCIIIdentifierStart(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentifierStart(input string) bool {
	if input == "" {
		return false
	}
	if input[0] < utf8.RuneSelf {
		return isASCIIIdentifierStart(input[0])
	}
	r, _ := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError && len(input) > 0 && input[0] < utf8.RuneSelf {
		r = rune(input[0])
	}
	return unicode.IsLetter(r) || r == '_' || r == '/'
}

func isIdentifierPart(input string) bool {
	if input == "" {
		return false
	}
	if input[0] < utf8.RuneSelf {
		return isASCIIIdentifierPart(input[0])
	}
	r, _ := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError && len(input) > 0 && input[0] < utf8.RuneSelf {
		r = rune(input[0])
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '/'
}

func isASCIIIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '/'
}

func isASCIIIdentifierPart(ch byte) bool {
	return isASCIIIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func (l *Lexer) advanceBytes(size int) {
	if size <= 0 {
		return
	}
	l.readPosition = l.position + size
	if l.readPosition >= len(l.input) {
		l.position = len(l.input)
		l.ch = 0
		return
	}
	l.position = l.readPosition
	l.ch = l.input[l.position]
	l.readPosition++
}
