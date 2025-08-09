package wanf

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
)

// --- Stream Lexer (from io.Reader) ---

// streamLexer 是一个从 io.Reader 读取数据的词法分析器.
// 它使用 bufio.Reader 来实现高效的预读(peek)功能, 并使用两个交替的 bytes.Buffer 来实现零分配的词法单元字面量生成.
type streamLexer struct {
	r       *bufio.Reader
	ch      byte
	line    int
	column  int
	bufA    bytes.Buffer
	bufB    bytes.Buffer
	useBufA bool
}

// newStreamLexer 创建一个新的流式词法分析器.
func newStreamLexer(r io.Reader) *streamLexer {
	l := &streamLexer{
		r:       bufio.NewReader(r),
		line:    1,
		useBufA: true,
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
				if bytes.Contains(literal, []byte{'.'}) {
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

// activeBuffer 返回当前用于构建字面量的缓冲区, 并在下次调用时切换到另一个.
// 这保证了在支持解析器两步预读的同时, 无需为每个词法单元分配新的内存.
func (l *streamLexer) activeBuffer() *bytes.Buffer {
	var buf *bytes.Buffer
	if l.useBufA {
		buf = &l.bufA
	} else {
		buf = &l.bufB
	}
	l.useBufA = !l.useBufA // 为下一个词法单元切换缓冲区
	buf.Reset()
	return buf
}

func (l *streamLexer) readDurationSuffix(prefix []byte) []byte {
	buf := l.activeBuffer()
	buf.Write(prefix)
	if l.ch == 'm' || l.ch == 'u' || l.ch == 'n' {
		if l.peekChar() == 's' {
			buf.WriteByte(l.ch)
			l.readChar()
		}
	}
	buf.WriteByte(l.ch)
	l.readChar()
	return buf.Bytes()
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
	buf := l.activeBuffer()
	for l.ch != '\n' && l.ch != 0 {
		buf.WriteByte(l.ch)
		l.readChar()
	}
	return buf.Bytes()
}

func (l *streamLexer) readMultiLineComment() ([]byte, bool) {
	buf := l.activeBuffer()
	startLine, startCol := l.line, l.column
	buf.WriteByte(l.ch)
	l.readChar()
	buf.WriteByte(l.ch)
	l.readChar()
	for {
		if l.ch == 0 {
			l.line, l.column = startLine, startCol
			return buf.Bytes(), false
		}
		if l.ch == '*' && l.peekChar() == '/' {
			buf.WriteByte(l.ch)
			l.readChar()
			buf.WriteByte(l.ch)
			l.readChar()
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		buf.WriteByte(l.ch)
		l.readChar()
	}
	return buf.Bytes(), true
}

func (l *streamLexer) readIdentifier() []byte {
	buf := l.activeBuffer()
	for isIdentifierChar(l.ch) {
		buf.WriteByte(l.ch)
		l.readChar()
	}
	return buf.Bytes()
}

func (l *streamLexer) readNumber() []byte {
	buf := l.activeBuffer()
	isFloat := false
	for unicode.IsDigit(rune(l.ch)) || (l.ch == '.' && !isFloat) {
		if l.ch == '.' {
			isFloat = true
		}
		buf.WriteByte(l.ch)
		l.readChar()
	}
	return buf.Bytes()
}

func (l *streamLexer) readString(quote byte) []byte {
	buf := l.activeBuffer()
	l.readChar()
	for {
		if l.ch == quote || l.ch == 0 {
			break
		}
		buf.WriteByte(l.ch)
		l.readChar()
	}
	l.readChar()
	return buf.Bytes()
}

func (l *streamLexer) readUntilEndOfLine() []byte {
	buf := l.activeBuffer()
	for l.ch != '\n' && l.ch != '\r' && l.ch != 0 {
		buf.WriteByte(l.ch)
		l.readChar()
	}
	return buf.Bytes()
}
