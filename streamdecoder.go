package wanf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"time"
)

// StreamDecoder 从输入流中读取并解码WANF格式的数据.
// 这是一个真正的流式解码器, 它边解析边解码, 不会为整个文件构建AST.
// 为了性能和低内存占用, 此解码器不支持 `var` 和 `import` 语句.
type StreamDecoder struct {
	d *internalDecoder
	p *Parser
}

// NewStreamDecoder 返回一个从 io.Reader 中读取数据的新解码器.
func NewStreamDecoder(r io.Reader, opts ...DecoderOption) (*StreamDecoder, error) {
	d := &internalDecoder{vars: make(map[string]interface{})}
	for _, opt := range opts {
		opt(d)
	}

	l := newStreamLexer(r)
	p := NewParser(l)

	dec := &StreamDecoder{
		d: d,
		p: p,
	}

	return dec, nil
}

// Decode reads and decodes the WANF stream into the value pointed to by v.
func (dec *StreamDecoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}

	err := dec.decodeBody(rv.Elem())
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

// decodeBody consumes tokens and decodes them into the reflect.Value.
func (dec *StreamDecoder) decodeBody(rv reflect.Value) error {
	for {
		if dec.p.curTokenIs(EOF) {
			return io.EOF
		}

		switch dec.p.curToken.Type {
		case SEMICOLON, COMMENT:
			dec.p.nextToken()
			continue
		case VAR:
			return fmt.Errorf("wanf: var statements are not supported in stream decoding mode (line %d)", dec.p.curToken.Line)
		case IMPORT:
			return fmt.Errorf("wanf: import statements are not supported in stream decoding mode (line %d)", dec.p.curToken.Line)
		case IDENT:
			if dec.p.peekTokenIs(ASSIGN) {
				if err := dec.decodeAssignStatement(rv); err != nil {
					return err
				}
			} else if dec.p.peekTokenIs(LBRACE) || dec.p.peekTokenIs(STRING) {
				if err := dec.decodeBlockStatement(rv); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("wanf: unexpected token %s after identifier %q on line %d", dec.p.peekToken.Type, dec.p.curToken.Literal, dec.p.curToken.Line)
			}
		case RBRACE:
			return nil
		default:
			return fmt.Errorf("wanf: unexpected token %s at top level on line %d", dec.p.curToken.Type, dec.p.curToken.Line)
		}

		dec.p.nextToken()
	}
}

// decodeAssignStatement decodes an assignment statement on the fly.
func (dec *StreamDecoder) decodeAssignStatement(rv reflect.Value) error {
	ident := dec.p.curToken

	if !dec.p.expectPeek(ASSIGN) {
		return fmt.Errorf("wanf: expected '=' after identifier %q", ident.Literal)
	}
	dec.p.nextToken()

	val, err := dec.evalExpressionOnTheFly()
	if err != nil {
		return err
	}

	field, tag, ok := findFieldAndTag(rv, ident.Literal)
	if !ok {
		return nil
	}

	if tag.KeyField != "" {
		return dec.d.setMapFromList(field, val, tag.KeyField)
	}
	return dec.d.setField(field, val)
}

// decodeBlockStatement decodes a block statement on the fly.
func (dec *StreamDecoder) decodeBlockStatement(rv reflect.Value) error {
	blockName := dec.p.curToken.Literal
	dec.p.nextToken()

	var label string
	if dec.p.curTokenIs(STRING) {
		label = string(dec.p.curToken.Literal)
		dec.p.nextToken()
	}

	if !dec.p.curTokenIs(LBRACE) {
		return fmt.Errorf("wanf: expected '{' after block identifier on line %d", dec.p.curToken.Line)
	}
	dec.p.nextToken()

	field, _, ok := findFieldAndTag(rv, blockName)
	if !ok {
		return dec.skipBlock()
	}

	switch field.Kind() {
	case reflect.Struct:
		if err := dec.decodeBody(field); err != nil {
			return err
		}
	case reflect.Map:
		if field.IsNil() {
			field.Set(reflect.MakeMap(field.Type()))
		}
		mapElemType := field.Type().Elem()
		newElem := reflect.New(mapElemType).Elem()
		if err := dec.decodeBody(newElem); err != nil {
			return err
		}
		if label == "" {
			return fmt.Errorf("wanf: map block %q requires a label", blockName)
		}
		field.SetMapIndex(reflect.ValueOf(label), newElem)

	default:
		return fmt.Errorf("wanf: block %q cannot be decoded into field of type %s", blockName, field.Type())
	}

	if !dec.p.curTokenIs(RBRACE) {
		return fmt.Errorf("wanf: expected '}' to close block %q on line %d", blockName, dec.p.curToken.Line)
	}
	return nil
}

// evalExpressionOnTheFly evaluates an expression by consuming tokens directly
// from the parser, without building an expression AST.
func (dec *StreamDecoder) evalExpressionOnTheFly() (interface{}, error) {
	switch dec.p.curToken.Type {
	case INT:
		return strconv.ParseInt(string(dec.p.curToken.Literal), 0, 64)
	case FLOAT:
		return strconv.ParseFloat(string(dec.p.curToken.Literal), 64)
	case STRING:
		return string(dec.p.curToken.Literal), nil
	case BOOL:
		return strconv.ParseBool(string(dec.p.curToken.Literal))
	case DUR:
		return time.ParseDuration(string(dec.p.curToken.Literal))
	case IDENT:
		// This can only be an `env()` call in this context.
		if bytes.Equal(dec.p.curToken.Literal, []byte("env")) {
			return dec.evalEnvExpressionOnTheFly()
		}
	case LBRACK:
		return dec.decodeListLiteralOnTheFly()
	case LBRACE:
		// This could be a block literal (like in a list) or a map literal `{[...`
		if dec.p.peekTokenIs(LBRACK) {
			return dec.decodeMapLiteralOnTheFly()
		}
		return dec.decodeBlockLiteralOnTheFly()
	}
	return nil, fmt.Errorf("wanf: unexpected token %s in expression", dec.p.curToken.Type)
}

func (dec *StreamDecoder) decodeListLiteralOnTheFly() (interface{}, error) {
	var list []interface{}
	dec.p.nextToken() // consume '['

	for !dec.p.curTokenIs(RBRACK) && !dec.p.curTokenIs(EOF) {
		val, err := dec.evalExpressionOnTheFly()
		if err != nil {
			return nil, err
		}
		list = append(list, val)
		dec.p.nextToken()

		if dec.p.curTokenIs(COMMA) {
			dec.p.nextToken() // consume comma
		} else if !dec.p.curTokenIs(RBRACK) {
			return nil, fmt.Errorf("wanf: expected comma or ']' in list literal")
		}
	}
	return list, nil
}

func (dec *StreamDecoder) decodeBlockLiteralOnTheFly() (interface{}, error) {
	m := make(map[string]interface{})
	dec.p.nextToken() // consume '{'

	for !dec.p.curTokenIs(RBRACE) && !dec.p.curTokenIs(EOF) {
		if dec.p.curTokenIs(COMMENT) || dec.p.curTokenIs(SEMICOLON) {
			dec.p.nextToken()
			continue
		}
		if !dec.p.curTokenIs(IDENT) {
			return nil, fmt.Errorf("wanf: expected identifier as key in block literal")
		}
		key := string(dec.p.curToken.Literal)

		if !dec.p.expectPeek(ASSIGN) {
			return nil, fmt.Errorf("wanf: expected '=' after key in block literal")
		}
		dec.p.nextToken() // consume value token

		val, err := dec.evalExpressionOnTheFly()
		if err != nil {
			return nil, err
		}
		m[key] = val
		dec.p.nextToken()
	}
	return m, nil
}

func (dec *StreamDecoder) decodeMapLiteralOnTheFly() (interface{}, error) {
	m := make(map[string]interface{})
	dec.p.nextToken() // consume '{'
	dec.p.nextToken() // consume '['

	for !dec.p.curTokenIs(RBRACK) && !dec.p.curTokenIs(EOF) {
		if !dec.p.curTokenIs(IDENT) {
			return nil, fmt.Errorf("wanf: expected identifier as key in map literal")
		}
		key := string(dec.p.curToken.Literal)
		if !dec.p.expectPeek(ASSIGN) {
			return nil, fmt.Errorf("wanf: expected '=' after key in map literal")
		}
		dec.p.nextToken() // consume value token
		val, err := dec.evalExpressionOnTheFly()
		if err != nil {
			return nil, err
		}
		m[key] = val
		dec.p.nextToken()

		if dec.p.curTokenIs(COMMA) {
			dec.p.nextToken()
		} else if !dec.p.curTokenIs(RBRACK) {
			return nil, fmt.Errorf("wanf: expected comma or ']' in map literal")
		}
	}
	dec.p.nextToken() // consume ']'
	if !dec.p.curTokenIs(RBRACE) {
		return nil, fmt.Errorf("wanf: expected '}' to close map literal")
	}
	return m, nil
}

func (dec *StreamDecoder) evalEnvExpressionOnTheFly() (interface{}, error) {
	if !dec.p.expectPeek(LPAREN) {
		return nil, fmt.Errorf("wanf: expected '(' after env")
	}
	dec.p.nextToken() // consume '('

	if !dec.p.curTokenIs(STRING) {
		return nil, fmt.Errorf("wanf: expected string argument for env()")
	}
	envVarName := string(dec.p.curToken.Literal)

	// Check for default value
	if dec.p.peekTokenIs(COMMA) {
		dec.p.nextToken() // consume ','
		dec.p.nextToken() // consume default value string token
		if !dec.p.curTokenIs(STRING) {
			return nil, fmt.Errorf("wanf: expected string for env() default value")
		}
		defaultValue := string(dec.p.curToken.Literal)
		if val, found := os.LookupEnv(envVarName); found {
			return val, nil
		}
		return defaultValue, nil
	}

	// No default value
	if val, found := os.LookupEnv(envVarName); found {
		return val, nil
	}

	if !dec.p.peekTokenIs(RPAREN) {
		return nil, fmt.Errorf("wanf: expected ')' after env() call")
	}
	dec.p.nextToken() // consume ')'

	return nil, fmt.Errorf("wanf: environment variable %q not set and no default provided", envVarName)
}

// skipBlock consumes tokens until the matching RBRACE is found.
func (dec *StreamDecoder) skipBlock() error {
	openBraces := 1
	for {
		if dec.p.curTokenIs(EOF) {
			return fmt.Errorf("wanf: unclosed block while skipping")
		}
		if dec.p.curTokenIs(LBRACE) {
			openBraces++
		}
		if dec.p.curTokenIs(RBRACE) {
			openBraces--
			if openBraces == 0 {
				break
			}
		}
		dec.p.nextToken()
	}
	return nil
}
