package expression

// Expr is the interface for all expression AST nodes.
type Expr interface {
	exprNode() // marker method
}

// FuncCall represents a function invocation: name(arg1, arg2, ...)
type FuncCall struct {
	Name string
	Args []Expr
}

// StringLit represents a string literal.
type StringLit struct {
	Value string
}

// NumberLit represents a numeric literal.
type NumberLit struct {
	Value float64
	IsInt bool
}

// BoolLit represents a boolean literal.
type BoolLit struct {
	Value bool
}

// VarRef represents a variable reference like "fixture.user.name".
type VarRef struct {
	Path string
}

func (*FuncCall) exprNode()  {}
func (*StringLit) exprNode() {}
func (*NumberLit) exprNode() {}
func (*BoolLit) exprNode()   {}
func (*VarRef) exprNode()    {}
