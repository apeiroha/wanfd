package wanf

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// StreamDecoder 从输入流中读取并解码WANF格式的数据.
type StreamDecoder struct {
	program *RootNode
	d       *internalDecoder
}

// NewStreamDecoder 返回一个从 io.Reader 中读取数据的新解码器.
// 解码器内部会进行缓冲, 它比标准的 Decode 函数在处理大文件时更节省内存,
// 因为它不会一次性将整个文件读入内存.
func NewStreamDecoder(r io.Reader, opts ...DecoderOption) (*StreamDecoder, error) {
	// Here we use the new streamLexer which reads from an io.Reader.
	l := newStreamLexer(r)
	p := NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		var errs []string
		for _, err := range p.Errors() {
			errs = append(errs, err.Error())
		}
		return nil, fmt.Errorf("parser errors: %s", strings.Join(errs, "\n"))
	}

	d := &internalDecoder{vars: make(map[string]interface{})}
	for _, opt := range opts {
		opt(d)
	}

	// The import and var processing logic is identical to the original decoder,
	// ensuring consistent behavior.
	finalStmts, err := processImports(program.Statements, d.basePath, make(map[string]bool))
	if err != nil {
		return nil, err
	}
	program.Statements = finalStmts

	for _, stmt := range program.Statements {
		if s, ok := stmt.(*VarStatement); ok {
			val, err := d.evalExpression(s.Value)
			if err != nil {
				return nil, err
			}
			d.vars[s.Name.Value] = val
		}
	}

	return &StreamDecoder{program: program, d: d}, nil
}

// Decode reads the next WANF-encoded value from its
// input and stores it in the value pointed to by v.
func (dec *StreamDecoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}
	// The core decoding logic is reused.
	return dec.d.decodeRoot(dec.program, rv.Elem())
}
