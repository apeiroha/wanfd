package wanf

import (
	"strings"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `
// config
var a = 1
import "path"

server "main" {
    host = "127.0.0.1" // host
    port = 8080
    timeout = 10s
    enabled = true
    features = [ "a", "b" ]
    /*
    block comment
    */
    rate = 0.5
    raw_data = ` + "`" + `line1
line2` + "`" + `
}
`
	// Normalize input to use \n for all line endings to make test deterministic.
	input = strings.ReplaceAll(input, "\r\n", "\n")

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{COMMENT, "// config"},
		{VAR, "var"},
		{IDENT, "a"},
		{ASSIGN, "="},
		{INT, "1"},
		{IMPORT, "import"},
		{STRING, "path"},
		{IDENT, "server"},
		{STRING, "main"},
		{LBRACE, "{"},
		{IDENT, "host"},
		{ASSIGN, "="},
		{STRING, "127.0.0.1"},
		{COMMENT, "// host"},
		{IDENT, "port"},
		{ASSIGN, "="},
		{INT, "8080"},
		{IDENT, "timeout"},
		{ASSIGN, "="},
		{DUR, "10s"},
		{IDENT, "enabled"},
		{ASSIGN, "="},
		{BOOL, "true"},
		{IDENT, "features"},
		{ASSIGN, "="},
		{LBRACK, "["},
		{STRING, "a"},
		{COMMA, ","},
		{STRING, "b"},
		{RBRACK, "]"},
		{COMMENT, "/*\n    block comment\n    */"},
		{IDENT, "rate"},
		{ASSIGN, "="},
		{FLOAT, "0.5"},
		{IDENT, "raw_data"},
		{ASSIGN, "="},
		{STRING, "line1\nline2"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := NewLexer([]byte(input))

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if string(tok.Literal) != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, string(tok.Literal))
		}
	}
}
