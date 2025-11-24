package expr

import (
	"testing"
)

func TestParseSimpleEquality(t *testing.T) {
	expr, err := ParseExpr("id = 1")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	binExpr, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", expr)
	}

	if binExpr.Op != OpEquals {
		t.Errorf("expected OpEquals, got %v", binExpr.Op)
	}

	col, ok := binExpr.Left.(*ColumnRefExpr)
	if !ok || col.Name != "id" {
		t.Errorf("expected column 'id', got %v", binExpr.Left)
	}

	lit, ok := binExpr.Right.(*LiteralExpr)
	if !ok || lit.Value != int64(1) {
		t.Errorf("expected literal 1, got %v", binExpr.Right)
	}
}

func TestParseComplexAnd(t *testing.T) {
	expr, err := ParseExpr("id > 10 AND status = 'active'")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	andExpr, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr for AND, got %T", expr)
	}

	if andExpr.Op != OpAnd {
		t.Errorf("expected OpAnd, got %v", andExpr.Op)
	}

	// Left: id > 10
	leftExpr, ok := andExpr.Left.(*BinaryExpr)
	if !ok || leftExpr.Op != OpGreaterThan {
		t.Errorf("left side should be 'id > 10', got %v", andExpr.Left)
	}

	// Right: status = 'active'
	rightExpr, ok := andExpr.Right.(*BinaryExpr)
	if !ok || rightExpr.Op != OpEquals {
		t.Errorf("right side should be 'status = active', got %v", andExpr.Right)
	}
}

func TestParseOrExpression(t *testing.T) {
	expr, err := ParseExpr("id = 1 OR id = 2")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	orExpr, ok := expr.(*BinaryExpr)
	if !ok || orExpr.Op != OpOr {
		t.Fatalf("expected OpOr, got %v", expr)
	}
}

func TestParseNot(t *testing.T) {
	expr, err := ParseExpr("NOT status = 'inactive'")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	notExpr, ok := expr.(*UnaryExpr)
	if !ok || notExpr.Op != OpNot {
		t.Fatalf("expected NOT expression, got %T", expr)
	}
}

func TestParseIsNull(t *testing.T) {
	expr, err := ParseExpr("email IS NULL")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	unaryExpr, ok := expr.(*UnaryExpr)
	if !ok || unaryExpr.Op != OpIsNull {
		t.Fatalf("expected IS NULL expression, got %T", expr)
	}
}

func TestParseIsNotNull(t *testing.T) {
	expr, err := ParseExpr("email IS NOT NULL")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	unaryExpr, ok := expr.(*UnaryExpr)
	if !ok || unaryExpr.Op != OpIsNotNull {
		t.Fatalf("expected IS NOT NULL expression, got %T", expr)
	}
}

func TestParseParentheses(t *testing.T) {
	expr, err := ParseExpr("(id = 1 OR id = 2) AND status = 'active'")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	andExpr, ok := expr.(*BinaryExpr)
	if !ok || andExpr.Op != OpAnd {
		t.Fatalf("expected AND at top level, got %v", expr)
	}

	// Left should be OR expression
	leftOr, ok := andExpr.Left.(*BinaryExpr)
	if !ok || leftOr.Op != OpOr {
		t.Errorf("left side should be OR, got %v", andExpr.Left)
	}
}

func TestParseStringLiteral(t *testing.T) {
	expr, err := ParseExpr("name = 'Alice'")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	binExpr, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", expr)
	}

	lit, ok := binExpr.Right.(*LiteralExpr)
	if !ok || lit.Value != "Alice" {
		t.Errorf("expected string 'Alice', got %v", binExpr.Right)
	}
}

func TestParseFloatLiteral(t *testing.T) {
	expr, err := ParseExpr("price > 99.99")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}

	binExpr, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", expr)
	}

	lit, ok := binExpr.Right.(*LiteralExpr)
	if !ok || lit.Value != 99.99 {
		t.Errorf("expected float 99.99, got %v", binExpr.Right)
	}
}

func TestParseEmptyString(t *testing.T) {
	expr, err := ParseExpr("")
	if err != nil {
		t.Fatalf("ParseExpr failed on empty string: %v", err)
	}

	if expr != nil {
		t.Errorf("expected nil for empty input, got %v", expr)
	}
}

func TestParseInvalidSyntax(t *testing.T) {
	tests := []string{
		"id = ",      // Missing right operand
		"= 1",        // Missing left operand
		"id 1",       // Missing operator
		"id = 1 AND", // Incomplete AND
		"(id = 1",    // Unclosed parenthesis
	}

	for _, sql := range tests {
		_, err := ParseExpr(sql)
		if err == nil {
			t.Errorf("expected error for invalid SQL: %q", sql)
		}
	}
}
