package wanf

// lexer 是一个对词法分析器行为进行抽象的接口.
// 原始的基于字节切片的 Lexer 和新的基于流的 streamLexer 都实现了此接口,
// 这使得解析器(Parser)可以无差别地使用它们.
type lexer interface {
	NextToken() Token
}
