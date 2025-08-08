package wanf

import (
	"bytes"
	"unicode"
)

type Lexer struct {
	input        []byte // 使用 []byte 避免复制
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func NewLexer(input []byte) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
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
	l.column++
}

func (l *Lexer) NextToken() Token {
	var tok Token
	l.skipWhitespace()
	line, col := l.line, l.column
	switch l.ch {
	case '=':
		tok = l.newToken(ASSIGN, l.ch, line, col)
	case ',':
		tok = l.newToken(COMMA, l.ch, line, col)
	case ';':
		tok = l.newToken(SEMICOLON, l.ch, line, col)
	case '{':
		tok = l.newToken(LBRACE, l.ch, line, col)
	case '}':
		tok = l.newToken(RBRACE, l.ch, line, col)
	case '[':
		tok = l.newToken(LBRACK, l.ch, line, col)
	case ']':
		tok = l.newToken(RBRACK, l.ch, line, col)
	case '(':
		tok = l.newToken(LPAREN, l.ch, line, col)
	case ')':
		tok = l.newToken(RPAREN, l.ch, line, col)
	case '#':
		tok.Type = ILLEGAL_COMMENT
		tok.Literal = l.readUntilEndOfLine()
		tok.Line = line
		tok.Column = col
		return tok
	case '$':
		if l.peekChar() == '{' {
			l.readChar()
			tok = Token{Type: DOLLAR_LBRACE, Literal: []byte("${"), Line: line, Column: col}
		} else {
			tok = l.newToken(ILLEGAL, l.ch, line, col)
		}
	case '"', '\'', '`':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = line
		tok.Column = col
		return tok
	case '/':
		if l.peekChar() == '/' {
			tok.Type = COMMENT
			tok.Literal = l.readSingleLineComment()
			tok.Line = line
			tok.Column = col
		} else if l.peekChar() == '*' {
			tok.Type = COMMENT
			tok.Literal = l.readMultiLineComment()
			tok.Line = line
			tok.Column = col
		} else {
			tok = l.newToken(ILLEGAL, l.ch, line, col)
			l.readChar()
		}
		return tok
	case 0:
		tok.Literal = []byte{}
		tok.Type = EOF
		l.readChar()
		return tok
	default:
		if isIdentifierStart(l.ch) {
			literal := l.readIdentifier()
			tok.Type = LookupIdentifier(literal)
			tok.Literal = literal
			tok.Line = line
			tok.Column = col
			return tok
		} else if unicode.IsDigit(rune(l.ch)) {
			literal := l.readNumber()
			if l.ch == 's' || l.ch == 'm' || l.ch == 'h' || (l.ch == 'u' && l.peekChar() == 's') || (l.ch == 'n' && l.peekChar() == 's') || (l.ch == 'm' && l.peekChar() == 's') {
				startPos := l.position - len(literal)
				l.readDurationSuffix()
				tok.Type = DUR
				tok.Literal = l.input[startPos:l.position]
			} else {
				if bytes.Contains(literal, []byte(".")) {
					tok.Type = FLOAT
				} else {
					tok.Type = INT
				}
				tok.Literal = literal
			}
			tok.Line = line
			tok.Column = col
			return tok
		} else {
			tok = l.newToken(ILLEGAL, l.ch, line, col)
		}
	}
	l.readChar()
	return tok
}

func (l *Lexer) readDurationSuffix() {
	if l.ch == 'm' || l.ch == 'u' || l.ch == 'n' {
		if l.peekChar() == 's' {
			l.readChar()
		}
	}
	l.readChar()
}
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}
func (l *Lexer) readSingleLineComment() []byte {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readMultiLineComment() []byte {
	position := l.position
	l.readChar() // consume '/'
	l.readChar() // consume '*'
	for {
		if l.ch == 0 {
			return l.input[position:l.position]
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readIdentifier() []byte {
	position := l.position
	for isIdentifierChar(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}
func (l *Lexer) readNumber() []byte {
	position := l.position
	isFloat := false
	for unicode.IsDigit(rune(l.ch)) || (l.ch == '.' && !isFloat) {
		if l.ch == '.' {
			isFloat = true
		}
		l.readChar()
	}
	return l.input[position:l.position]
}
func (l *Lexer) readString() []byte {
	quote := l.ch
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == quote || l.ch == 0 {
			break
		}
	}
	literal := l.input[position:l.position]
	l.readChar()
	return literal
}

func (l *Lexer) readUntilEndOfLine() []byte {
	position := l.position
	for {
		if l.ch == '\n' || l.ch == '\r' || l.ch == 0 {
			break
		}
		l.readChar()
	}
	return l.input[position:l.position]
}
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}
func (l *Lexer) newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: []byte{ch}, Line: line, Column: column}
}
func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}
func isIdentifierChar(ch byte) bool {
	return isIdentifierStart(ch) || unicode.IsDigit(rune(ch))
}
