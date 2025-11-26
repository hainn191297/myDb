package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/expr"
)

// evaluateFilter evaluates a filter expression against a row and returns true if it matches.
// It expects the expression to evaluate to a boolean.
func evaluateFilter(ctx context.Context, filter expr.Expr, row Row, tableDef *schema.TableDef) (bool, error) {
	val, err := evaluateExpr(ctx, filter, row, tableDef)
	if err != nil {
		return false, err
	}
	if val == nil {
		return false, nil // NULL is false in WHERE
	}
	if b, ok := val.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("filter expression must evaluate to bool, got %T", val)
}

func evaluateExpr(ctx context.Context, e expr.Expr, row Row, tableDef *schema.TableDef) (any, error) {
	switch n := e.(type) {
	case *expr.BinaryExpr:
		return evaluateBinary(ctx, n, row, tableDef)
	case *expr.UnaryExpr:
		return evaluateUnary(ctx, n, row, tableDef)
	case *expr.LiteralExpr:
		return n.Value, nil
	case *expr.ColumnRefExpr:
		return resolveColumn(n.Name, row, tableDef)
	default:
		return nil, fmt.Errorf("unknown expression type: %T", e)
	}
}

func resolveColumn(name string, row Row, tableDef *schema.TableDef) (any, error) {
	// Find column index in row
	idx := -1
	for i, colName := range row.Columns {
		if strings.EqualFold(colName, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("column %s not found in row", name)
	}

	// Get raw value
	rawVal := row.Values[idx]
	if rawVal == nil {
		return nil, nil // NULL
	}

	// Find column definition to get type
	if tableDef == nil {
		// Fallback for raw mode (no schema) - treat as bytes or string?
		// For MVP, assume string if no schema
		return string(rawVal), nil
	}

	var colDef *schema.ColumnDef
	for _, col := range tableDef.Columns {
		if strings.EqualFold(col.Name, name) {
			colDef = &col
			break
		}
	}
	if colDef == nil {
		return nil, fmt.Errorf("column %s not found in table definition", name)
	}

	// Decode value
	return colDef.Type.Decode(rawVal)
}

func evaluateBinary(ctx context.Context, n *expr.BinaryExpr, row Row, tableDef *schema.TableDef) (any, error) {
	left, err := evaluateExpr(ctx, n.Left, row, tableDef)
	if err != nil {
		return nil, err
	}
	right, err := evaluateExpr(ctx, n.Right, row, tableDef)
	if err != nil {
		return nil, err
	}

	// Handle NULLs
	if left == nil || right == nil {
		return nil, nil
	}

	switch n.Op {
	case expr.OpEquals:
		return compare(left, right) == 0, nil
	case expr.OpNotEquals:
		return compare(left, right) != 0, nil
	case expr.OpLessThan:
		return compare(left, right) < 0, nil
	case expr.OpLessOrEqual:
		return compare(left, right) <= 0, nil
	case expr.OpGreaterThan:
		return compare(left, right) > 0, nil
	case expr.OpGreaterOrEqual:
		return compare(left, right) >= 0, nil
	case expr.OpAnd:
		l, ok1 := left.(bool)
		r, ok2 := right.(bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("AND requires boolean operands")
		}
		return l && r, nil
	case expr.OpOr:
		l, ok1 := left.(bool)
		r, ok2 := right.(bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("OR requires boolean operands")
		}
		return l || r, nil
	default:
		return nil, fmt.Errorf("unknown binary operator: %v", n.Op)
	}
}

func evaluateUnary(ctx context.Context, n *expr.UnaryExpr, row Row, tableDef *schema.TableDef) (any, error) {
	val, err := evaluateExpr(ctx, n.Expr, row, tableDef)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case expr.OpNot:
		if val == nil {
			return nil, nil // NOT NULL is NULL
		}
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("NOT requires boolean operand")
		}
		return !b, nil
	case expr.OpIsNull:
		return val == nil, nil
	case expr.OpIsNotNull:
		return val != nil, nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %v", n.Op)
	}
}

func compare(a, b any) int {
	switch va := a.(type) {
	case int64:
		if vb, ok := b.(int64); ok {
			if va < vb {
				return -1
			}
			if va > vb {
				return 1
			}
			return 0
		}
		if vb, ok := b.(int); ok {
			vbi := int64(vb)
			if va < vbi {
				return -1
			}
			if va > vbi {
				return 1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			if va < vb {
				return -1
			}
			if va > vb {
				return 1
			}
			return 0
		}
	case bool:
		if vb, ok := b.(bool); ok {
			if va == vb {
				return 0
			}
			if !va && vb {
				return -1
			} // false < true
			return 1
		}
	case float64:
		if vb, ok := b.(float64); ok {
			if va < vb {
				return -1
			}
			if va > vb {
				return 1
			}
			return 0
		}
	}

	// Fallback: string comparison
	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}
