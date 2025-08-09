package wanf

import (
	"fmt"
	"io"
	"reflect"
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

	exprAST := dec.p.parseExpression(LOWEST)
	if exprAST == nil {
		return fmt.Errorf("wanf: could not parse expression for key %q", ident.Literal)
	}
	val, err := dec.d.evalExpression(exprAST)
	if err != nil {
		return err
	}

	field, tag, ok := findFieldAndTag(rv, string(ident.Literal))
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

	field, _, ok := findFieldAndTag(rv, string(blockName))
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
