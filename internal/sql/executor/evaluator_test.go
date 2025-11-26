package executor

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/expr"
)

func TestEvaluateFilterBasic(t *testing.T) {
	ctx := context.Background()

	// Create table definition
	tableDef := &schema.TableDef{
		Schema: "public",
		Table:  "users",
		Columns: []schema.ColumnDef{
			{Name: "id", Type: schema.TypeInt64},
			{Name: "age", Type: schema.TypeInt64},
			{Name: "name", Type: schema.TypeText},
		},
	}

	// Encode test values
	idVal, _ := schema.TypeInt64.Encode(int64(1))
	ageVal, _ := schema.TypeInt64.Encode(int64(25))
	nameVal, _ := schema.TypeText.Encode("Alice")

	row := Row{
		Columns: []string{"id", "age", "name"},
		Values:  [][]byte{idVal, ageVal, nameVal},
	}

	// Test: id = 1
	filterExpr, _ := expr.ParseExpr("id = 1")
	match, err := evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if !match {
		t.Error("expected id = 1 to match")
	}

	// Test: age > 20
	filterExpr, _ = expr.ParseExpr("age > 20")
	match, err = evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if !match {
		t.Error("expected age > 20 to match")
	}

	// Test: age < 20
	filterExpr, _ = expr.ParseExpr("age < 20")
	match, err = evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if match {
		t.Error("expected age < 20 to NOT match")
	}

	// Test: name = 'Alice'
	filterExpr, _ = expr.ParseExpr("name = 'Alice'")
	match, err = evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if !match {
		t.Error("expected name = 'Alice' to match")
	}

	// Test: age >= 25 AND name = 'Alice'
	filterExpr, _ = expr.ParseExpr("age >= 25 AND name = 'Alice'")
	match, err = evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if !match {
		t.Error("expected complex AND to match")
	}

	// Test: age > 30 OR name = 'Alice'
	filterExpr, _ = expr.ParseExpr("age > 30 OR name = 'Alice'")
	match, err = evaluateFilter(ctx, filterExpr, row, tableDef)
	if err != nil {
		t.Fatalf("evaluateFilter failed: %v", err)
	}
	if !match {
		t.Error("expected complex OR to match")
	}
}
