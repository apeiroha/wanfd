package wanf

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
)

// This file contains the stream-based lexer.

// streamLexer 是一个从 io.Reader 读取数据的词法分析器.
type streamLexer struct {
	r      *bufio.Reader
	ch     byte
	line   int
	column int
	// Reusable buffer for building literals.
	literalBuf bytes.Buffer
}

// newStreamLexer creates a new stream-based lexer.
func newStreamLexer(r io.Reader) *streamLexer {
	l := &streamLexer{
		r:    bufio.NewReader(r),
		line: 1,
	}
	l.readChar()
	return l
}

func (l *streamLexer) readChar() {
	var err error
	l.ch, err = l.r.ReadByte()
	if err != nil {
		l.ch = 0
	}
	l.column++
}

func (l *streamLexer) peekChar() byte {
	b, err := l.r.Peek(1)
	if err != nil {
		return 0
	}
	return b[0]
}

func (l *streamLexer) newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{Type: tokenType, Literal: singleCharByteSlices[ch], Line: line, Column: column}
}

func (l *streamLexer) NextToken() Token {
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
		quote := l.ch
		tok.Type = STRING
		tok.Literal = l.readString(quote)
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
			literal, ok := l.readMultiLineComment()
			if !ok {
				tok.Type = ILLEGAL
				tok.Literal = []byte("unclosed block comment")
			} else {
				tok.Type = COMMENT
				tok.Literal = literal
			}
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
				tok.Type = DUR
				tok.Literal = l.readDurationSuffix(literal)
			} else {
				if bytes.Contains(literal, dot) {
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

func (l *streamLexer) readDurationSuffix(prefix []byte) []byte {
	l.literalBuf.Reset()
	l.literalBuf.Write(prefix)
	if l.ch == 'm' || l.ch == 'u' || l.ch == 'n' {
		if l.peekChar() == 's' {
			l.literalBuf.WriteByte(l.ch)
			l.readChar()
		}
	}
	l.literalBuf.WriteByte(l.ch)
	l.readChar()
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}

func (l *streamLexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *streamLexer) readSingleLineComment() []byte {
	l.literalBuf.Reset()
	for l.ch != '\n' && l.ch != 0 {
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}

func (l *streamLexer) readMultiLineComment() ([]byte, bool) {
	l.literalBuf.Reset()
	startLine, startCol := l.line, l.column
	l.literalBuf.WriteByte(l.ch)
	l.readChar()
	l.literalBuf.WriteByte(l.ch)
	l.readChar()
	for {
		if l.ch == 0 {
			l.line, l.column = startLine, startCol
			return l.literalBuf.Bytes(), false
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.literalBuf.WriteByte(l.ch)
			l.readChar()
			l.literalBuf.WriteByte(l.ch)
			l.readChar()
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c, true
}

func (l *streamLexer) readIdentifier() []byte {
	l.literalBuf.Reset()
	for isIdentifierChar(l.ch) {
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}

func (l *streamLexer) readNumber() []byte {
	l.literalBuf.Reset()
	isFloat := false
	for unicode.IsDigit(rune(l.ch)) || (l.ch == '.' && !isFloat) {
		if l.ch == '.' {
			isFloat = true
		}
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}

func (l *streamLexer) readString(quote byte) []byte {
	l.literalBuf.Reset()
	l.readChar()
	for {
		if l.ch == quote || l.ch == 0 {
			break
		}
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	l.readChar()
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}

func (l *streamLexer) readUntilEndOfLine() []byte {
	l.literalBuf.Reset()
	for l.ch != '\n' && l.ch != '\r' && l.ch != 0 {
		l.literalBuf.WriteByte(l.ch)
		l.readChar()
	}
	c := make([]byte, l.literalBuf.Len())
	copy(c, l.literalBuf.Bytes())
	return c
}
