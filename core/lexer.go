package core

import (
	"strings"
	"unicode"
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
			ch := l.ch
			l.readChar()
			tok = Token{Type: EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(ASSIGN, l.ch)
		}
	case ':':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: COLON_ASSIGN, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(ILLEGAL, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(PIPE, l.ch)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(AMPERSAND, l.ch)
		}
	case '>':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: APPEND, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(GREATER, l.ch)
		}
	case '<':
		tok = newToken(LESS, l.ch)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: NEQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = newToken(NOT, l.ch)
		}
	case '+':
		tok = newToken(PLUS, l.ch)
	case ';':
		tok = newToken(SEMICOLON, l.ch)
	case ',':
		tok = newToken(COMMA, l.ch)
	case '.':
		tok = newToken(DOT, l.ch)
	case '(':
		tok = newToken(LPAREN, l.ch)
	case ')':
		tok = newToken(RPAREN, l.ch)
	case '{':
		tok = newToken(LBRACE, l.ch)
	case '}':
		tok = newToken(RBRACE, l.ch)
	case '$':
		tok = newToken(DOLLAR, l.ch)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
	case 0:
		if l.isCompletable() {
			tok = Token{Type: SEMICOLON, Literal: ";"}
			l.prevToken = SEMICOLON
			return tok
		}
		tok.Literal = ""
		tok.Type = EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			l.prevToken = tok.Type
			return tok
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			l.prevToken = tok.Type
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch)
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
	for {
		if isLetter(l.ch) || isDigit(l.ch) {
			l.readChar()
		} else if l.ch == '=' && (isLetter(l.peekChar()) || isDigit(l.peekChar())) {
			l.readChar()
		} else {
			break
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

func (l *Lexer) readString() string {
	position := l.position + 1
	var out strings.Builder
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
		if l.ch == '\\' {
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
		} else {
			out.WriteByte(l.ch)
		}
	}
	if out.Len() > 0 {
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

func newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Literal: string(ch)}
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_' || ch == '-' || ch == '/'
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
