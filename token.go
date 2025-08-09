package wanf

import (
	"bytes"
	"fmt"
)

type TokenType string

type Token struct {
	Type    TokenType
	Literal []byte // 使用 []byte 避免在词法分析阶段分配新字符串
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("Line:%d, Col:%d, Type:%s, Literal:`%s`", t.Line, t.Column, t.Type, string(t.Literal))
}

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	IDENT   TokenType = "IDENT"
	INT     TokenType = "INT"
	FLOAT   TokenType = "FLOAT"
	STRING  TokenType = "STRING"
	BOOL    TokenType = "BOOL"
	DUR     TokenType = "DUR"
	ASSIGN  TokenType = "="
	COMMA   TokenType = ","
	SEMICOLON TokenType = ";"
	LBRACE  TokenType = "{"
	RBRACE  TokenType = "}"
	LBRACK  TokenType = "["
	RBRACK  TokenType = "]"
	LPAREN  TokenType = "("
	RPAREN  TokenType = ")"
	IMPORT  TokenType = "IMPORT"
	VAR     TokenType = "VAR"
	DOLLAR_LBRACE TokenType = "${"
	COMMENT TokenType = "COMMENT"
	ILLEGAL_COMMENT TokenType = "ILLEGAL_COMMENT"
)

// LookupIdentifier 检查 ident 是否是关键字.
// 使用 bytes.Equal 可以实现零内存分配的关键字匹配.
func LookupIdentifier(ident []byte) TokenType {
	if bytes.Equal(ident, []byte("import")) {
		return IMPORT
	}
	if bytes.Equal(ident, []byte("var")) {
		return VAR
	}
	if bytes.Equal(ident, []byte("true")) {
		return BOOL
	}
	if bytes.Equal(ident, []byte("false")) {
		return BOOL
	}
	return IDENT
}
