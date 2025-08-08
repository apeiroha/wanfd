package wanf

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var encoderPool = sync.Pool{
	New: func() interface{} {
		return &internalEncoder{
			buf: &bytes.Buffer{},
		}
	},
}

// fieldCache caches processed field information for a given struct type.
var fieldCache sync.Map // map[reflect.Type][]cachedField

func getEncoder() *internalEncoder {
	return encoderPool.Get().(*internalEncoder)
}

func putEncoder(e *internalEncoder) {
	e.buf.Reset()
	e.indent = 0
	encoderPool.Put(e)
}

func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type EncoderOption func(*FormatOptions)

func WithStyle(style OutputStyle) EncoderOption {
	return func(o *FormatOptions) {
		o.Style = style
	}
}

func WithoutEmptyLines() EncoderOption {
	return func(o *FormatOptions) {
		o.EmptyLines = false
	}
}

type Encoder struct {
	w io.Writer
	e *internalEncoder
}

func NewEncoder(w io.Writer, opts ...EncoderOption) *Encoder {
	options := FormatOptions{
		Style:      StyleDefault,
		EmptyLines: true,
	}
	for _, opt := range opts {
		opt(&options)
	}
	e := getEncoder()
	e.opts = options
	return &Encoder{w: w, e: e}
}

func (enc *Encoder) Encode(v interface{}) error {
	defer putEncoder(enc.e)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return fmt.Errorf("wanf: can only encode a non-nil struct")
	}
	if err := enc.e.encodeStruct(rv, 0); err != nil {
		return err
	}
	if enc.e.opts.Style != StyleSingleLine && enc.e.buf.Len() > 0 {
		enc.e.buf.WriteString("\n")
	}
	_, err := enc.w.Write(enc.e.buf.Bytes())
	return err
}

type internalEncoder struct {
	buf    *bytes.Buffer
	indent int
	opts   FormatOptions
}

type fieldInfo struct {
	name      string
	value     reflect.Value
	tag       wanfTag
	fieldType reflect.StructField
	isBlock   bool
}

type cachedField struct {
	name      string
	tag       wanfTag
	fieldType reflect.StructField
	isBlock   bool
	index     int
}

func (e *internalEncoder) encodeStruct(v reflect.Value, depth int) error {
	fields := e.gatherFields(v)
	if e.opts.Style == StyleDefault && depth > 0 {
		sort.Slice(fields, func(i, j int) bool {
			if fields[i].isBlock != fields[j].isBlock {
				return !fields[i].isBlock
			}
			return fields[i].name < fields[j].name
		})
	}

	var prevWasBlock bool
	for i, f := range fields {
		e.writeSeparator(i > 0, f.isBlock, prevWasBlock)
		e.encodeField(f, depth)
		prevWasBlock = f.isBlock
	}
	return nil
}

func (e *internalEncoder) encodeField(f fieldInfo, depth int) {
	e.writeIndent()
	e.buf.WriteString(f.name)
	e.writeSpace()

	if f.isBlock {
		e.buf.WriteString("{")
		e.writeNewLine()
		e.indent++
		if f.value.Kind() == reflect.Map {
			e.encodeMap(f.value, depth+1)
		} else {
			e.encodeStruct(f.value, depth+1)
		}
		e.indent--
		e.writeNewLine()
		e.writeIndent()
		e.buf.WriteString("}")
	} else {
		e.buf.WriteString("=")
		e.writeSpace()
		e.encodeValue(f.value, depth)
	}
}

func (e *internalEncoder) encodeValue(v reflect.Value, depth int) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if d, ok := v.Interface().(time.Duration); ok {
		e.buf.WriteString(d.String())
		return
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if e.opts.Style != StyleSingleLine && strings.Contains(s, "\n") {
			e.buf.WriteString("`" + s + "`")
		} else {
			e.buf.WriteString(strconv.Quote(s))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.buf.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Float32, reflect.Float64:
		e.buf.WriteString(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case reflect.Bool:
		e.buf.WriteString(strconv.FormatBool(v.Bool()))
	case reflect.Slice, reflect.Array:
		e.encodeSlice(v, depth)
	case reflect.Struct:
		e.buf.WriteString("{")
		e.writeNewLine()
		e.indent++
		e.encodeStruct(v, depth+1)
		e.indent--
		e.writeNewLine()
		e.writeIndent()
		e.buf.WriteString("}")
	case reflect.Map:
		e.encodeMap(v, depth)
	}
}

func (e *internalEncoder) encodeSlice(v reflect.Value, depth int) {
	e.buf.WriteString("[")
	l := v.Len()
	if l == 0 {
		e.buf.WriteString("]")
		return
	}
	e.writeNewLine()
	e.indent++
	for i := 0; i < l; i++ {
		e.writeIndent()
		e.encodeValue(v.Index(i), depth)
		if i < l-1 {
			e.buf.WriteString(",")
		}
		e.writeNewLine()
	}
	e.indent--
	e.writeIndent()
	e.buf.WriteString("]")
}

func (e *internalEncoder) encodeMap(v reflect.Value, depth int) {
	keys := make([]string, 0, v.Len())
	for _, key := range v.MapKeys() {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			if e.opts.Style == StyleSingleLine {
				e.buf.WriteString(";")
			} else {
				e.writeNewLine()
			}
		}
		e.writeIndent()
		e.buf.WriteString(key)
		e.writeSpace()
		e.buf.WriteString("=")
		e.writeSpace()
		e.encodeValue(v.MapIndex(reflect.ValueOf(key)), depth)
	}
}

func (e *internalEncoder) writeIndent() {
	if e.opts.Style != StyleSingleLine {
		for i := 0; i < e.indent; i++ {
			e.buf.WriteByte('\t')
		}
	}
}
func (e *internalEncoder) writeNewLine() {
	if e.opts.Style != StyleSingleLine {
		e.buf.WriteString("\n")
	}
}
func (e *internalEncoder) writeSpace() {
	if e.opts.Style != StyleSingleLine {
		e.buf.WriteString(" ")
	}
}
func (e *internalEncoder) writeSeparator(isNotFirst, isCurrentBlock, isPrevBlock bool) {
	if !isNotFirst {
		return
	}
	if e.opts.Style == StyleSingleLine {
		e.buf.WriteString(";")
		return
	}
	e.writeNewLine()
	if e.opts.Style == StyleDefault && e.opts.EmptyLines && (isCurrentBlock || isPrevBlock) {
		e.writeNewLine()
	}
}

func (e *internalEncoder) gatherFields(v reflect.Value) []fieldInfo {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	cached, ok := fieldCache.Load(t)
	if !ok {
		cached = cacheStructInfo(t)
		fieldCache.Store(t, cached)
	}

	cachedFields := cached.([]cachedField)
	fields := make([]fieldInfo, 0, len(cachedFields))
	for _, cf := range cachedFields {
		fieldVal := v.Field(cf.index)
		if (cf.tag.Omitempty && isZero(fieldVal)) || (fieldVal.Kind() == reflect.Map && fieldVal.Len() == 0) {
			continue
		}
		fields = append(fields, fieldInfo{
			name:      cf.name,
			value:     fieldVal,
			tag:       cf.tag,
			fieldType: cf.fieldType,
			isBlock:   cf.isBlock,
		})
	}
	return fields
}

func cacheStructInfo(t reflect.Type) []cachedField {
	var cachedFields []cachedField
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		tagStr := fieldType.Tag.Get("wanf")
		tagInfo := parseWanfTag(tagStr, fieldType.Name)
		cachedFields = append(cachedFields, cachedField{
			name:      tagInfo.Name,
			tag:       tagInfo,
			fieldType: fieldType,
			isBlock:   isBlockType(fieldType.Type, tagInfo),
			index:     i,
		})
	}
	return cachedFields
}

func isBlockType(ft reflect.Type, tag wanfTag) bool {
	if ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	isStruct := ft.Kind() == reflect.Struct && ft.Name() != "Duration"
	isMap := ft.Kind() == reflect.Map && tag.KeyField == ""
	return isStruct || isMap
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	}
	return false
}
