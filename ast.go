package wanf

import (
	"bytes"
	"strings"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// Node 是AST中所有节点的基础接口.
type Node interface {
	TokenLiteral() string
	String() string
	Format(w *bytes.Buffer, indent string, opts FormatOptions)
}

// Statement 代表一个语句.
type Statement interface {
	Node
	statementNode()
	GetLeadingComments() []*Comment
}

// Expression 代表一个表达式.
type Expression interface {
	Node
	expressionNode()
}

// Comment 表示一个注释节点
type Comment struct {
	Token Token
	Text  string
}

func (c *Comment) expressionNode()      {}
func (c *Comment) statementNode()       {}
func (c *Comment) TokenLiteral() string { return string(c.Token.Literal) }
func (c *Comment) String() string       { return c.Text }
func (c *Comment) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.WriteString(c.Text)
}

// RootNode 是每个WANF文件AST的根节点.
type RootNode struct {
	Statements []Statement
}

func (p *RootNode) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *RootNode) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	p.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}

func (p *RootNode) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	// 辅助函数, 用于判断语句类型和是否有注释
	isBlock := func(s Statement) bool {
		_, ok := s.(*BlockStatement)
		return ok
	}
	hasComments := func(s Statement) bool {
		return len(s.GetLeadingComments()) > 0
	}

	for i, s := range p.Statements {
		if i > 0 {
			if opts.Style == StyleSingleLine {
				w.WriteString("; ")
			} else {
				w.WriteString("\n")
				// 启发式规则: 如果上一个或当前是块, 或者当前有注释, 则增加一个空行
				if opts.EmptyLines && (isBlock(p.Statements[i-1]) || isBlock(s) || hasComments(s)) {
					w.WriteString("\n")
				}
			}
		}
		s.Format(w, indent, opts)
	}
}

// --- 语句 (Statements) ---

// AssignStatement 表示一个赋值语句, 如 `key = value`.
type AssignStatement struct {
	Token           Token
	Name            *Identifier
	Value           Expression
	LeadingComments []*Comment // 前置注释
	LineComment     *Comment   // 行尾注释
}

func (as *AssignStatement) statementNode() {}
func (as *AssignStatement) GetLeadingComments() []*Comment {
	return as.LeadingComments
}
func (as *AssignStatement) TokenLiteral() string { return string(as.Token.Literal) }
func (as *AssignStatement) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	as.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (as *AssignStatement) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	for _, c := range as.LeadingComments {
		w.WriteString(indent)
		w.WriteString(c.Text)
		w.WriteString("\n")
	}
	w.WriteString(indent)
	as.Name.Format(w, indent, opts)
	w.WriteString(" = ")
	if as.Value != nil {
		as.Value.Format(w, indent, opts)
	}
	if as.LineComment != nil {
		w.WriteString(" ")
		w.WriteString(as.LineComment.Text)
	}
}

// BlockStatement 表示一个块, 如 `database { ... }`.
type BlockStatement struct {
	Token           Token
	Name            *Identifier
	Label           *StringLiteral
	Body            *RootNode
	LeadingComments []*Comment // 前置注释
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) GetLeadingComments() []*Comment {
	return bs.LeadingComments
}
func (bs *BlockStatement) TokenLiteral() string { return string(bs.Token.Literal) }
func (bs *BlockStatement) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	bs.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (bs *BlockStatement) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	for _, c := range bs.LeadingComments {
		w.WriteString(indent)
		w.WriteString(c.Text)
		w.WriteString("\n")
	}
	w.WriteString(indent)
	bs.Name.Format(w, indent, opts)
	if bs.Label != nil {
		w.WriteString(" ")
		bs.Label.Format(w, indent, opts)
	}
	if opts.Style == StyleSingleLine {
		w.WriteString("{")
		bs.Body.Format(w, "", opts)
		w.WriteString("}")
	} else {
		w.WriteString(" {")
		if len(bs.Body.Statements) > 0 {
			w.WriteString("\n")
			bs.Body.Format(w, indent+"\t", opts)
		}
		w.WriteString("\n" + indent + "}")
	}
}

// VarStatement 表示一个变量声明, 如 `var name = value`.
type VarStatement struct {
	Token           Token
	Name            *Identifier
	Value           Expression
	LeadingComments []*Comment // 前置注释
	LineComment     *Comment   // 行尾注释
}

func (vs *VarStatement) statementNode() {}
func (vs *VarStatement) GetLeadingComments() []*Comment {
	return vs.LeadingComments
}
func (vs *VarStatement) TokenLiteral() string { return string(vs.Token.Literal) }
func (vs *VarStatement) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	vs.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (vs *VarStatement) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	for _, c := range vs.LeadingComments {
		w.WriteString(indent)
		w.WriteString(c.Text)
		w.WriteString("\n")
	}
	w.WriteString(indent)
	w.WriteString(vs.TokenLiteral() + " ")
	vs.Name.Format(w, indent, opts)
	w.WriteString(" = ")
	if vs.Value != nil {
		vs.Value.Format(w, indent, opts)
	}
	if vs.LineComment != nil {
		w.WriteString(" ")
		w.WriteString(vs.LineComment.Text)
	}
}

// ImportStatement 表示一个导入语句, 如 `import "path/to/file.wanf"`.
type ImportStatement struct {
	Token           Token
	Path            *StringLiteral
	LeadingComments []*Comment // 前置注释
	LineComment     *Comment   // 行尾注释
}

func (is *ImportStatement) statementNode() {}
func (is *ImportStatement) GetLeadingComments() []*Comment {
	return is.LeadingComments
}
func (is *ImportStatement) TokenLiteral() string { return string(is.Token.Literal) }
func (is *ImportStatement) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	is.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (is *ImportStatement) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	for _, c := range is.LeadingComments {
		w.WriteString(indent)
		w.WriteString(c.Text)
		w.WriteString("\n")
	}
	w.WriteString(indent)
	w.WriteString(is.TokenLiteral() + " ")
	is.Path.Format(w, indent, opts)
	if is.LineComment != nil {
		w.WriteString(" ")
		w.WriteString(is.LineComment.Text)
	}
}

// --- 表达式 (Expressions) ---

// Identifier 表示一个标识符.
type Identifier struct {
	Token Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return string(i.Token.Literal) }
func (i *Identifier) String() string       { return i.Value }
func (i *Identifier) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.WriteString(i.Value)
}

// Literal 表示一个字面量.
type Literal interface {
	Expression
	literalNode()
}

// StringLiteral 表示一个字符串字面量.
type StringLiteral struct {
	Token Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) literalNode()         {}
func (sl *StringLiteral) TokenLiteral() string { return string(sl.Token.Literal) }
func (sl *StringLiteral) String() string {
	if strings.Contains(sl.Value, "\n") {
		return "`" + sl.Value + "`"
	}
	return `"` + sl.Value + `"`
}
func (sl *StringLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	if opts.Style != StyleSingleLine && strings.Contains(sl.Value, "\n") {
		w.WriteString("`" + sl.Value + "`")
	} else {
		w.WriteString(`"`)
		w.WriteString(sl.Value)
		w.WriteString(`"`)
	}
}

// IntegerLiteral 表示一个整数.
type IntegerLiteral struct {
	Token Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) literalNode()         {}
func (il *IntegerLiteral) TokenLiteral() string { return string(il.Token.Literal) }
func (il *IntegerLiteral) String() string       { return string(il.Token.Literal) }
func (il *IntegerLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.Write(il.Token.Literal)
}

// FloatLiteral 表示一个浮点数.
type FloatLiteral struct {
	Token Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) literalNode()         {}
func (fl *FloatLiteral) TokenLiteral() string { return string(fl.Token.Literal) }
func (fl *FloatLiteral) String() string       { return string(fl.Token.Literal) }
func (fl *FloatLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.Write(fl.Token.Literal)
}

// BoolLiteral 表示一个布尔值.
type BoolLiteral struct {
	Token Token
	Value bool
}

func (bl *BoolLiteral) expressionNode()      {}
func (bl *BoolLiteral) literalNode()         {}
func (bl *BoolLiteral) TokenLiteral() string { return string(bl.Token.Literal) }
func (bl *BoolLiteral) String() string       { return string(bl.Token.Literal) }
func (bl *BoolLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.Write(bl.Token.Literal)
}

// DurationLiteral 表示一个持续时间.
type DurationLiteral struct {
	Token Token
	Value string
}

func (dl *DurationLiteral) expressionNode()      {}
func (dl *DurationLiteral) literalNode()         {}
func (dl *DurationLiteral) TokenLiteral() string { return string(dl.Token.Literal) }
func (dl *DurationLiteral) String() string       { return string(dl.Token.Literal) }
func (dl *DurationLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.Write(dl.Token.Literal)
}

// ListLiteral 表示一个列表, 如 `[el1, el2]`.
type ListLiteral struct {
	Token            Token
	Elements         []Expression
	HasTrailingComma bool
}

func (ll *ListLiteral) expressionNode()      {}
func (ll *ListLiteral) TokenLiteral() string { return string(ll.Token.Literal) }
func (ll *ListLiteral) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	ll.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (ll *ListLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	if opts.Style == StyleSingleLine {
		w.WriteString("[")
		for i, el := range ll.Elements {
			if i > 0 {
				w.WriteString(",")
			}
			el.Format(w, "", opts)
		}
		w.WriteString("]")
	} else {
		w.WriteString("[\n")
		newIndent := indent + "\t"
		for i, el := range ll.Elements {
			if i > 0 {
				w.WriteString(",\n")
			}
			w.WriteString(newIndent)
			el.Format(w, newIndent, opts)
		}
		w.WriteString("\n" + indent + "]")
	}
}

// MapLiteral 表示一个映射字面量, 如 `{[ key = val ]}`.
type MapLiteral struct {
	Token    Token // The '{[' token
	Elements []*AssignStatement
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return string(ml.Token.Literal) }
func (ml *MapLiteral) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	ml.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (ml *MapLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	if opts.Style == StyleSingleLine {
		w.WriteString("{[")
		for i, el := range ml.Elements {
			el.Format(w, "", opts)
			if i < len(ml.Elements)-1 {
				w.WriteString(",")
			}
		}
		w.WriteString("]}")
	} else {
		w.WriteString("{[\n")
		newIndent := indent + "\t"
		for _, el := range ml.Elements {
			w.WriteString(newIndent)
			el.Format(w, newIndent, opts)
			w.WriteString(",\n")
		}
		w.WriteString(indent + "]}")
	}
}

// BlockLiteral 表示一个匿名的块, 通常用作值, 例如在列表中.
type BlockLiteral struct {
	Token Token
	Body  *RootNode
}

func (bl *BlockLiteral) expressionNode()      {}
func (bl *BlockLiteral) TokenLiteral() string { return string(bl.Token.Literal) }
func (bl *BlockLiteral) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	bl.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (bl *BlockLiteral) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	if opts.Style == StyleSingleLine {
		w.WriteString("{")
		bl.Body.Format(w, "", opts)
		w.WriteString("}")
	} else {
		w.WriteString("{\n")
		bl.Body.Format(w, indent+"\t", opts)
		w.WriteString("\n" + indent + "}")
	}
}

// VarExpression 表示一个变量引用, 如 `${var}`.
type VarExpression struct {
	Token Token
	Name  string
}

func (ve *VarExpression) expressionNode()      {}
func (ve *VarExpression) TokenLiteral() string { return string(ve.Token.Literal) }
func (ve *VarExpression) String() string       { return "${" + ve.Name + "}" }
func (ve *VarExpression) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.WriteString("${" + ve.Name + "}")
}

// EnvExpression 表示对 `env()` 函数的调用.
type EnvExpression struct {
	Token        Token
	Name         *StringLiteral
	DefaultValue *StringLiteral
}

func (ee *EnvExpression) expressionNode()      {}
func (ee *EnvExpression) TokenLiteral() string { return string(ee.Token.Literal) }
func (ee *EnvExpression) String() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()
	ee.Format(buf, "", FormatOptions{Style: StyleDefault, EmptyLines: true})
	return buf.String()
}
func (ee *EnvExpression) Format(w *bytes.Buffer, indent string, opts FormatOptions) {
	w.WriteString("env(")
	ee.Name.Format(w, indent, opts)
	if ee.DefaultValue != nil {
		w.WriteString(", ")
		ee.DefaultValue.Format(w, indent, opts)
	}
	w.WriteString(")")
}
