package expr

// Expr is the interface for all expression nodes in the WHERE clause AST.
type Expr interface {
	exprNode() // marker method
}

// ValueType represents the type of a literal value.
type ValueType int

const (
	TypeInt ValueType = iota
	TypeText
	TypeBool
	TypeFloat
	TypeNull
)

// LiteralExpr represents a constant value (123, 'hello', true, 45.67).
type LiteralExpr struct {
	Value any // int64, string, bool, float64
	Type  ValueType
}

func (*LiteralExpr) exprNode() {}

// ColumnRefExpr represents a column reference (id, email, status).
type ColumnRefExpr struct {
	Name string
}

func (*ColumnRefExpr) exprNode() {}

// BinaryOp represents binary operators.
type BinaryOp int

const (
	// Comparison operators
	OpEquals BinaryOp = iota
	OpNotEquals
	OpLessThan
	OpLessOrEqual
	OpGreaterThan
	OpGreaterOrEqual

	// Logical operators
	OpAnd
	OpOr
)

func (op BinaryOp) String() string {
	switch op {
	case OpEquals:
		return "="
	case OpNotEquals:
		return "!="
	case OpLessThan:
		return "<"
	case OpLessOrEqual:
		return "<="
	case OpGreaterThan:
		return ">"
	case OpGreaterOrEqual:
		return ">="
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	default:
		return "UNKNOWN"
	}
}

// BinaryExpr represents binary operations (a = b, x > y, p AND q).
type BinaryExpr struct {
	Left  Expr
	Op    BinaryOp
	Right Expr
}

func (*BinaryExpr) exprNode() {}

// UnaryOp represents unary operators.
type UnaryOp int

const (
	OpNot UnaryOp = iota
	OpIsNull
	OpIsNotNull
)

func (op UnaryOp) String() string {
	switch op {
	case OpNot:
		return "NOT"
	case OpIsNull:
		return "IS NULL"
	case OpIsNotNull:
		return "IS NOT NULL"
	default:
		return "UNKNOWN"
	}
}

// UnaryExpr represents unary operations (NOT x, col IS NULL).
type UnaryExpr struct {
	Op   UnaryOp
	Expr Expr
}

func (*UnaryExpr) exprNode() {}

// Walk traverses an expression tree in depth-first order.
func Walk(e Expr, visitor func(Expr)) {
	if e == nil {
		return
	}
	visitor(e)
	switch n := e.(type) {
	case *BinaryExpr:
		Walk(n.Left, visitor)
		Walk(n.Right, visitor)
	case *UnaryExpr:
		Walk(n.Expr, visitor)
	}
}
