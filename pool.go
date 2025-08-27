package wanf

import "sync"

// AST Node Pools
var (
	assignStatementPool = sync.Pool{New: func() interface{} { return new(AssignStatement) }}
	blockStatementPool  = sync.Pool{New: func() interface{} { return new(BlockStatement) }}
	varStatementPool    = sync.Pool{New: func() interface{} { return new(VarStatement) }}
	importStatementPool = sync.Pool{New: func() interface{} { return new(ImportStatement) }}
	commentPool         = sync.Pool{New: func() interface{} { return new(Comment) }}
	identifierPool      = sync.Pool{New: func() interface{} { return new(Identifier) }}
	stringLiteralPool   = sync.Pool{New: func() interface{} { return new(StringLiteral) }}
	integerLiteralPool  = sync.Pool{New: func() interface{} { return new(IntegerLiteral) }}
	floatLiteralPool    = sync.Pool{New: func() interface{} { return new(FloatLiteral) }}
	boolLiteralPool     = sync.Pool{New: func() interface{} { return new(BoolLiteral) }}
	durationLiteralPool = sync.Pool{New: func() interface{} { return new(DurationLiteral) }}
	listLiteralPool     = sync.Pool{New: func() interface{} { return new(ListLiteral) }}
	blockLiteralPool    = sync.Pool{New: func() interface{} { return new(BlockLiteral) }}
	mapLiteralPool      = sync.Pool{New: func() interface{} { return new(MapLiteral) }}
	varExpressionPool   = sync.Pool{New: func() interface{} { return new(VarExpression) }}
	envExpressionPool   = sync.Pool{New: func() interface{} { return new(EnvExpression) }}
	rootNodePool        = sync.Pool{New: func() interface{} { return new(RootNode) }}
)

// Getter functions for AST nodes
func getAssignStatement() *AssignStatement {
	n := assignStatementPool.Get().(*AssignStatement)
	n.Reset()
	return n
}

func getBlockStatement() *BlockStatement {
	n := blockStatementPool.Get().(*BlockStatement)
	n.Reset()
	return n
}

func getVarStatement() *VarStatement {
	n := varStatementPool.Get().(*VarStatement)
	n.Reset()
	return n
}

func getImportStatement() *ImportStatement {
	n := importStatementPool.Get().(*ImportStatement)
	n.Reset()
	return n
}

func getComment() *Comment {
	n := commentPool.Get().(*Comment)
	n.Reset()
	return n
}

func getIdentifier() *Identifier {
	n := identifierPool.Get().(*Identifier)
	n.Reset()
	return n
}

func getStringLiteral() *StringLiteral {
	n := stringLiteralPool.Get().(*StringLiteral)
	n.Reset()
	return n
}

func getIntegerLiteral() *IntegerLiteral {
	n := integerLiteralPool.Get().(*IntegerLiteral)
	n.Reset()
	return n
}

func getFloatLiteral() *FloatLiteral {
	n := floatLiteralPool.Get().(*FloatLiteral)
	n.Reset()
	return n
}

func getBoolLiteral() *BoolLiteral {
	n := boolLiteralPool.Get().(*BoolLiteral)
	n.Reset()
	return n
}

func getDurationLiteral() *DurationLiteral {
	n := durationLiteralPool.Get().(*DurationLiteral)
	n.Reset()
	return n
}

func getListLiteral() *ListLiteral {
	n := listLiteralPool.Get().(*ListLiteral)
	n.Reset()
	return n
}

func getBlockLiteral() *BlockLiteral {
	n := blockLiteralPool.Get().(*BlockLiteral)
	n.Reset()
	return n
}

func getMapLiteral() *MapLiteral {
	n := mapLiteralPool.Get().(*MapLiteral)
	n.Reset()
	return n
}

func getVarExpression() *VarExpression {
	n := varExpressionPool.Get().(*VarExpression)
	n.Reset()
	return n
}

func getEnvExpression() *EnvExpression {
	n := envExpressionPool.Get().(*EnvExpression)
	n.Reset()
	return n
}

func getRootNode() *RootNode {
	n := rootNodePool.Get().(*RootNode)
	n.Reset()
	return n
}
