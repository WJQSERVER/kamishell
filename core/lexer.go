package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int       // current position in input (points to current char)
	readPosition int       // current reading position in input (after current char)
	ch           byte      // current char under examination
	prevToken    TokenType // type of the last token returned
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
			tok = newToken(ASSIGN, l.ch, l.position)
		}
	case ':':
		if l.peekChar() == '=' {
			start := l.position
			l.readChar()
			tok = Token{Type: COLON_ASSIGN, Literal: ":=", Start: start, End: l.readPosition}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.position)
		}
	case '|':
		if l.peekChar() == '|' {
			start := l.position
			l.readChar()
			tok = Token{Type: OR, Literal: "||", Start: start, End: l.readPosition}
		} else {
			tok = newToken(PIPE, l.ch, l.position)
		}
	case '&':
		if l.peekChar() == '&' {
			start := l.position
			l.readChar()
			tok = Token{Type: AND, Literal: "&&", Start: start, End: l.readPosition}
		} else {
			tok = newToken(AMPERSAND, l.ch, l.position)
		}
	case '>':
		if l.peekChar() == '>' {
			start := l.position
			l.readChar()
			tok = Token{Type: APPEND, Literal: ">>", Start: start, End: l.readPosition}
		} else {
			tok = newToken(GREATER, l.ch, l.position)
		}
	case '<':
		tok = newToken(LESS, l.ch, l.position)
	case '!':
		if l.peekChar() == '=' {
			start := l.position
			l.readChar()
			tok = Token{Type: NEQ, Literal: "!=", Start: start, End: l.readPosition}
		} else {
			tok = newToken(NOT, l.ch, l.position)
		}
	case '+':
		tok = newToken(PLUS, l.ch, l.position)
	case ';':
		tok = newToken(SEMICOLON, l.ch, l.position)
	case ',':
		tok = newToken(COMMA, l.ch, l.position)
	case '.':
		tok = newToken(DOT, l.ch, l.position)
	case '(':
		tok = newToken(LPAREN, l.ch, l.position)
	case ')':
		tok = newToken(RPAREN, l.ch, l.position)
	case '{':
		tok = newToken(LBRACE, l.ch, l.position)
	case '}':
		tok = newToken(RBRACE, l.ch, l.position)
	case '$':
		tok = newToken(DOLLAR, l.ch, l.position)
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
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			tok.Start = start
			tok.End = l.position
			l.prevToken = tok.Type
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch, l.position)
		}
	}

	l.readChar()
	l.prevToken = tok.Type
	return tok
}

func (l *Lexer) isCompletable() bool {
	switch l.prevToken {
	case IDENT, NUMBER, STRING, TRUE_TOK, FALSE_TOK, NIL, RETURN, RPAREN, RBRACE:
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
	for isIdentifierPart(l.input[l.position:]) {
		_, size := utf8.DecodeRuneInString(l.input[l.position:])
		if size <= 0 {
			break
		}
		l.advanceBytes(size)
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

func newToken(tokenType TokenType, ch byte, start int) Token {
	return Token{Type: tokenType, Literal: singleByteLiteral(ch), Start: start, End: start + 1}
}

func isLetter(ch byte) bool {
	return isIdentifierStart(string([]byte{ch}))
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentifierStart(input string) bool {
	if input == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError && len(input) > 0 && input[0] < utf8.RuneSelf {
		r = rune(input[0])
	}
	return unicode.IsLetter(r) || r == '_' || r == '-' || r == '/'
}

func isIdentifierPart(input string) bool {
	if input == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(input)
	if r == utf8.RuneError && len(input) > 0 && input[0] < utf8.RuneSelf {
		r = rune(input[0])
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '/'
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

func singleByteLiteral(ch byte) string {
	switch ch {
	case '=':
		return "="
	case '|':
		return "|"
	case '>':
		return ">"
	case '<':
		return "<"
	case '&':
		return "&"
	case '!':
		return "!"
	case '+':
		return "+"
	case ';':
		return ";"
	case ',':
		return ","
	case '.':
		return "."
	case '(':
		return "("
	case ')':
		return ")"
	case '{':
		return "{"
	case '}':
		return "}"
	case '$':
		return "$"
	case ':':
		return ":"
	case '/':
		return "/"
	case '"':
		return "\""
	default:
		return string(ch)
	}
}
