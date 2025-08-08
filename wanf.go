package wanf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var varRegex = regexp.MustCompile(`\$\{(\w+)\}`)

func Lint(data []byte) (*RootNode, []LintError) {
	l := NewLexer(data)
	p := NewParser(l)
	p.SetLintMode(true)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return program, []LintError{
			{
				Line:      1,
				Column:    1,
				EndLine:   1,
				EndColumn: 1,
				Message:   "parser error: " + strings.Join(p.Errors(), "; "),
			},
		}
	}
	allErrors := p.LintErrors()
	analyzer := &astAnalyzer{
		errors:       allErrors,
		blockCounts:  make(map[string]int),
		declaredVars: make(map[string]*VarStatement),
		usedVars:     make(map[string]bool),
	}
	newProgram := analyzer.Analyze(program)
	return newProgram.(*RootNode), analyzer.errors
}

func Format(program *RootNode, opts FormatOptions) []byte {
	var out bytes.Buffer
	program.Format(&out, "", opts)
	return out.Bytes()
}

type astAnalyzer struct {
	errors       []LintError
	blockCounts  map[string]int
	declaredVars map[string]*VarStatement
	usedVars     map[string]bool
}

func (a *astAnalyzer) Analyze(node Node) Node {
	// First pass: collect block counts and declared variables.
	a.collect(node)

	// Second pass: check for issues.
	newNode := a.check(node)

	// Post-pass: check for unused variables.
	for name, stmt := range a.declaredVars {
		if _, ok := a.usedVars[name]; !ok {
			err := LintError{
				Line:      stmt.Token.Line,
				Column:    stmt.Token.Column,
				EndLine:   stmt.Token.Line,
				EndColumn: stmt.Token.Column + len(name),
				Message:   fmt.Sprintf("variable %q is declared but not used", name),
			}
			a.errors = append(a.errors, err)
		}
	}
	return newNode
}

func (a *astAnalyzer) collect(root Node) {
	if root == nil {
		return
	}
	stack := []Node{root}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if node == nil {
			continue
		}

		// Process the node
		switch n := node.(type) {
		case *BlockStatement:
			a.blockCounts[n.Name.Value]++
		case *VarStatement:
			a.declaredVars[n.Name.Value] = n
		}

		// Push children onto the stack
		switch n := node.(type) {
		case *RootNode:
			for i := len(n.Statements) - 1; i >= 0; i-- {
				stack = append(stack, n.Statements[i])
			}
		case *BlockStatement:
			stack = append(stack, n.Body)
		case *BlockLiteral:
			stack = append(stack, n.Body)
		case *AssignStatement:
			stack = append(stack, n.Value)
		case *ListLiteral:
			for i := len(n.Elements) - 1; i >= 0; i-- {
				stack = append(stack, n.Elements[i])
			}
		case *VarStatement:
			stack = append(stack, n.Value)
		}
	}
}

func (a *astAnalyzer) check(node Node) Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *RootNode:
		for i, stmt := range n.Statements {
			n.Statements[i] = a.check(stmt).(Statement)
		}
		return n
	case *BlockStatement:
		if n.Body != nil {
			n.Body = a.check(n.Body).(*RootNode)
		}
		if n.Label != nil && a.blockCounts[n.Name.Value] == 1 {
			err := LintError{
				Line:      n.Token.Line,
				Column:    n.Token.Column,
				EndLine:   n.Token.Line,
				EndColumn: n.Token.Column + len(n.Name.Value),
				Message:   fmt.Sprintf("block %q is defined only once, the label %q is redundant", n.Name.Value, n.Label.Value),
			}
			a.errors = append(a.errors, err)
			return &BlockStatement{
				Token:           n.Token,
				Name:            n.Name,
				Label:           nil,
				Body:            n.Body,
				LeadingComments: n.LeadingComments,
			}
		}
		return n
	case *BlockLiteral:
		if n.Body != nil {
			n.Body = a.check(n.Body).(*RootNode)
		}
		return n
	case *AssignStatement:
		if n.Value != nil {
			n.Value = a.check(n.Value).(Expression)
		}
		return n
	case *ListLiteral:
		for i, el := range n.Elements {
			n.Elements[i] = a.check(el).(Expression)
		}
		return n
	case *VarStatement:
		if n.Value != nil {
			n.Value = a.check(n.Value).(Expression)
		}
		return n
	case *VarExpression:
		a.usedVars[n.Name] = true
		return n
	case *StringLiteral:
		matches := varRegex.FindAllStringSubmatch(n.Value, -1)
		for _, match := range matches {
			if len(match) > 1 {
				a.usedVars[match[1]] = true
			}
		}
		return n
	default:
		return node
	}
}

type DecoderOption func(*internalDecoder)

func WithBasePath(path string) DecoderOption {
	return func(d *internalDecoder) {
		d.basePath = path
	}
}

type Decoder struct {
	program *RootNode
	d       *internalDecoder
}

func NewDecoder(r io.Reader, opts ...DecoderOption) (*Decoder, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	l := NewLexer(data)
	p := NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parser errors: %s", strings.Join(p.Errors(), "\n"))
	}
	d := &internalDecoder{vars: make(map[string]interface{})}
	for _, opt := range opts {
		opt(d)
	}
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
	return &Decoder{program: program, d: d}, nil
}

func processImports(stmts []Statement, basePath string, processed map[string]bool) ([]Statement, error) {
	var finalStmts []Statement
	for _, stmt := range stmts {
		importStmt, ok := stmt.(*ImportStatement)
		if !ok {
			finalStmts = append(finalStmts, stmt)
			continue
		}
		importPath := filepath.Join(basePath, importStmt.Path.Value)
		absImportPath, err := filepath.Abs(importPath)
		if err != nil {
			return nil, fmt.Errorf("could not get absolute path for import %q: %w", importPath, err)
		}
		if processed[absImportPath] {
			continue
		}
		processed[absImportPath] = true
		data, err := os.ReadFile(absImportPath)
		if err != nil {
			return nil, fmt.Errorf("could not read imported file %q: %w", importPath, err)
		}
		l := NewLexer(data)
		p := NewParser(l)
		program := p.ParseProgram()
		if len(p.Errors()) > 0 {
			return nil, fmt.Errorf("parser errors in imported file %q: %s", importPath, strings.Join(p.Errors(), "\n"))
		}
		importedStmts, err := processImports(program.Statements, filepath.Dir(absImportPath), processed)
		if err != nil {
			return nil, err
		}
		finalStmts = append(finalStmts, importedStmts...)
	}
	return finalStmts, nil
}

func (dec *Decoder) Decode(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}
	return dec.d.decodeRoot(dec.program, rv.Elem())
}

func DecodeFile(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec, err := NewDecoder(f, WithBasePath(filepath.Dir(path)))
	if err != nil {
		return err
	}
	return dec.Decode(v)
}

func Decode(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	// Since NewDecoder expects a Reader, we use bytes.NewReader here.
	// The internal implementation of NewDecoder will read the bytes and then
	// correctly pass the resulting []byte slice to NewLexer.
	dec, err := NewDecoder(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return dec.Decode(v)
}

type internalDecoder struct {
	vars     map[string]interface{}
	basePath string
}

func (d *internalDecoder) decodeRoot(root *RootNode, rv reflect.Value) error {
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("can only decode root into a struct, got %s", rv.Kind())
	}
	for _, stmt := range root.Statements {
		switch s := stmt.(type) {
		case *AssignStatement:
			if err := d.decodeAssign(s, rv); err != nil {
				return err
			}
		case *BlockStatement:
			if err := d.decodeBlock(s, rv); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *internalDecoder) decodeAssign(stmt *AssignStatement, rv reflect.Value) error {
	field, tag, ok := findFieldAndTag(rv, stmt.Name.Value)
	if !ok {
		return nil
	}
	val, err := d.evalExpression(stmt.Value)
	if err != nil {
		return err
	}
	if tag.KeyField != "" {
		return d.setMapFromList(field, val, tag.KeyField)
	}
	return d.setField(field, val)
}

func (d *internalDecoder) decodeBlock(stmt *BlockStatement, rv reflect.Value) error {
	field, _, ok := findFieldAndTag(rv, stmt.Name.Value)
	if !ok {
		return nil
	}
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return d.decodeRoot(stmt.Body, field.Elem())
	}
	if field.Kind() == reflect.Struct {
		return d.decodeRoot(stmt.Body, field)
	}
	if field.Kind() == reflect.Map {
		mapType := field.Type()
		if mapType.Key().Kind() == reflect.String && mapType.Elem().Kind() == reflect.String {
			if field.IsNil() {
				field.Set(reflect.MakeMap(mapType))
			}
			return d.decodeMapStringString(stmt.Body, field)
		}
		if stmt.Label == nil {
			return fmt.Errorf("block %q is for a map, but is missing a label", stmt.Name.Value)
		}
		mapVal := field
		if mapVal.IsNil() {
			mapVal.Set(reflect.MakeMap(mapVal.Type()))
		}
		elemType := mapVal.Type().Elem()
		newStruct := reflect.New(elemType).Elem()
		if err := d.decodeRoot(stmt.Body, newStruct); err != nil {
			return err
		}
		mapVal.SetMapIndex(reflect.ValueOf(stmt.Label.Value), newStruct)
	}
	return nil
}

func (d *internalDecoder) setField(field reflect.Value, val interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set field")
	}
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return d.setField(field.Elem(), val)
	}
	v := reflect.ValueOf(val)
	if v.Type().ConvertibleTo(field.Type()) {
		field.Set(v.Convert(field.Type()))
		return nil
	}
	if field.Kind() == reflect.Slice && v.Kind() == reflect.Slice {
		return d.setSliceField(field, v)
	}
	return fmt.Errorf("cannot set field of type %s with value of type %T", field.Type(), val)
}

func (d *internalDecoder) setSliceField(field, v reflect.Value) error {
	sliceType := field.Type()
	elemType := sliceType.Elem()
	newSlice := reflect.MakeSlice(sliceType, v.Len(), v.Len())
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i).Interface()
		valV := reflect.ValueOf(val)
		if valV.Type().ConvertibleTo(elemType) {
			newSlice.Index(i).Set(valV.Convert(elemType))
		} else {
			return fmt.Errorf("cannot convert slice element of type %T to %s", val, elemType)
		}
	}
	field.Set(newSlice)
	return nil
}

func (d *internalDecoder) evalExpression(expr Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *IntegerLiteral:
		return e.Value, nil
	case *FloatLiteral:
		return e.Value, nil
	case *StringLiteral:
		return e.Value, nil
	case *BoolLiteral:
		return e.Value, nil
	case *DurationLiteral:
		return time.ParseDuration(e.Value)
	case *VarExpression:
		val, ok := d.vars[e.Name]
		if !ok {
			return nil, fmt.Errorf("variable %q is not defined", e.Name)
		}
		return val, nil
	case *EnvExpression:
		val, found := os.LookupEnv(e.Name.Value)
		if !found {
			if e.DefaultValue != nil {
				return e.DefaultValue.Value, nil
			}
			return nil, fmt.Errorf("environment variable %q not set", e.Name.Value)
		}
		return val, nil
	case *ListLiteral:
		list := make([]interface{}, len(e.Elements))
		for i, elemExpr := range e.Elements {
			val, err := d.evalExpression(elemExpr)
			if err != nil {
				return nil, err
			}
			list[i] = val
		}
		return list, nil
	case *BlockLiteral:
		return d.decodeBlockToMap(e.Body)
	}
	return nil, fmt.Errorf("unknown expression type: %T", expr)
}

func (d *internalDecoder) decodeBlockToMap(body *RootNode) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	for _, stmt := range body.Statements {
		switch s := stmt.(type) {
		case *AssignStatement:
			val, err := d.evalExpression(s.Value)
			if err != nil {
				return nil, err
			}
			m[s.Name.Value] = val
		case *BlockStatement:
			nestedMap, err := d.decodeBlockToMap(s.Body)
			if err != nil {
				return nil, err
			}
			m[s.Name.Value] = nestedMap
		}
	}
	return m, nil
}

func findFieldAndTag(structVal reflect.Value, name string) (reflect.Value, wanfTag, bool) {
	typ := structVal.Type()

	// First pass: match by `wanf:"..."` tag
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagStr := field.Tag.Get("wanf")
		if tagStr != "" {
			tag := parseWanfTag(tagStr, field.Name)
			if tag.Name == name {
				return structVal.Field(i), tag, true
			}
		}
	}

	// Second pass: case-insensitive match by field name for fields without a tag
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagStr := field.Tag.Get("wanf")
		if tagStr == "" {
			if strings.EqualFold(field.Name, name) {
				tag := parseWanfTag("", field.Name)
				return structVal.Field(i), tag, true
			}
		}
	}

	return reflect.Value{}, wanfTag{}, false
}

func (d *internalDecoder) setMapFromList(mapField reflect.Value, listVal interface{}, keyField string) error {
	if mapField.Kind() != reflect.Map {
		return fmt.Errorf("cannot set list to non-map field %s", mapField.Type())
	}
	sourceList, ok := listVal.([]interface{})
	if !ok {
		return fmt.Errorf("value for map field with 'key' tag must be a list")
	}
	if mapField.IsNil() {
		mapField.Set(reflect.MakeMap(mapField.Type()))
	}
	elemType := mapField.Type().Elem()
	for _, item := range sourceList {
		sourceMap, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("items in list for keyed map must be objects")
		}
		keyVal, ok := sourceMap[keyField]
		if !ok {
			return fmt.Errorf("key field %q not found in list item", keyField)
		}
		keyString, ok := keyVal.(string)
		if !ok {
			return fmt.Errorf("key field %q must be a string", keyField)
		}
		newStruct := reflect.New(elemType).Elem()
		if err := d.decodeMapToStruct(sourceMap, newStruct); err != nil {
			return err
		}
		mapField.SetMapIndex(reflect.ValueOf(keyString), newStruct)
	}
	return nil
}

func (d *internalDecoder) decodeMapToStruct(sourceMap map[string]interface{}, targetStruct reflect.Value) error {
	for key, val := range sourceMap {
		field, _, ok := findFieldAndTag(targetStruct, key)
		if !ok {
			continue
		}
		if err := d.setField(field, val); err != nil {
			return fmt.Errorf("error setting field %q: %w", key, err)
		}
	}
	return nil
}

func (d *internalDecoder) decodeMapStringString(body *RootNode, mapField reflect.Value) error {
	for _, stmt := range body.Statements {
		assign, ok := stmt.(*AssignStatement)
		if !ok {
			continue
		}
		val, err := d.evalExpression(assign.Value)
		if err != nil {
			return err
		}
		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("value for key %q in map must be a string", assign.Name.Value)
		}
		mapField.SetMapIndex(reflect.ValueOf(assign.Name.Value), reflect.ValueOf(strVal))
	}
	return nil
}
