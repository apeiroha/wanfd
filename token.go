package wanf

import "fmt"

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

var keywordMap = map[string]TokenType{
	"import": IMPORT,
	"var":    VAR,
	"true":   BOOL,
	"false":  BOOL,
}

// LookupIdentifier 检查 ident 是否是关键字
func LookupIdentifier(ident []byte) TokenType {
	// 为了在map中查找,这里存在一个从[]byte到string的转换,会产生一次内存分配.
	// 但这是必要的,因为直接使用[]byte作为map的键是不安全的,除非能保证它们不会被修改.
	if tok, ok := keywordMap[string(ident)]; ok {
		return tok
	}
	return IDENT
}
