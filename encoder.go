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
		Style:      StyleBlockSorted,
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

	// 根据编码风格对字段进行排序
	// Sort fields based on the encoding style.
	switch e.opts.Style {
	case StyleBlockSorted, StyleAllSorted:
		// StyleBlockSorted只在嵌套层级排序, StyleAllSorted在所有层级排序
		// StyleBlockSorted only sorts in nested levels, StyleAllSorted sorts at all levels.
		if e.opts.Style == StyleAllSorted || depth > 0 {
			sort.Slice(fields, func(i, j int) bool {
				// isBlock为false的是kv对, true的是嵌套块. kv对优先.
				// Fields where isBlock is false are key-value pairs, true are nested blocks. KV pairs come first.
				if fields[i].isBlock != fields[j].isBlock {
					return !fields[i].isBlock
				}
				// 同类型按名称字母顺序排序
				// Same types are sorted alphabetically by name.
				return fields[i].name < fields[j].name
			})
		}
	case StyleStreaming:
		// StyleStreaming不进行任何排序, 保持结构体定义顺序
		// StyleStreaming does no sorting, preserving the struct definition order.
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
		if f.value.Kind() == reflect.Map {
			e.encodeMap(f.value, depth+1)
		} else {
			e.buf.WriteString("{")
			e.writeNewLine()
			e.indent++
			e.encodeStruct(f.value, depth+1)
			e.indent--
			e.writeNewLine()
			e.writeIndent()
			e.buf.WriteString("}")
		}
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
		// 处理空结构体, 紧凑输出: item = {}
		// Handle empty struct for compact output: item = {}
		if v.NumField() == 0 {
			e.buf.WriteString("{}")
			return
		}
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

	if e.opts.Style == StyleSingleLine {
		for i := 0; i < l; i++ {
			if i > 0 {
				e.buf.WriteString(",")
			}
			e.encodeValue(v.Index(i), depth)
		}
	} else {
		e.writeNewLine()
		e.indent++
		for i := 0; i < l; i++ {
			e.writeIndent()
			e.encodeValue(v.Index(i), depth)
			e.buf.WriteString(",")
			e.writeNewLine()
		}
		e.indent--
		e.writeIndent()
	}
	e.buf.WriteString("]")
}

func (e *internalEncoder) encodeMap(v reflect.Value, depth int) {
	e.buf.WriteString("{[")
	keys := v.MapKeys()
	if len(keys) == 0 {
		e.buf.WriteString("]}")
		return
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	if e.opts.Style == StyleSingleLine {
		for i, key := range keys {
			if i > 0 {
				e.buf.WriteString(",")
			}
			e.buf.WriteString(key.String())
			e.buf.WriteString("=")
			e.encodeValue(v.MapIndex(key), depth)
		}
	} else {
		e.writeNewLine()
		e.indent++
		for _, key := range keys {
			e.writeIndent()
			e.buf.WriteString(key.String())
			e.writeSpace()
			e.buf.WriteString("=")
			e.writeSpace()
			e.encodeValue(v.MapIndex(key), depth)
			e.buf.WriteString(",")
			e.writeNewLine()
		}
		e.indent--
		e.writeIndent()
	}
	e.buf.WriteString("]}")
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
	// 在StyleBlockSorted或StyleAllSorted模式下, 且启用了空行时, 在块之间添加空行
	// In StyleBlockSorted or StyleAllSorted mode, when empty lines are enabled, add an empty line between blocks.
	if (e.opts.Style == StyleBlockSorted || e.opts.Style == StyleAllSorted) && e.opts.EmptyLines && (isCurrentBlock || isPrevBlock) {
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
	// 只有结构体是块. 映射被视为值.
	// Only structs are blocks. Maps are treated as values.
	isStruct := ft.Kind() == reflect.Struct && ft.Name() != "Duration"
	return isStruct
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
