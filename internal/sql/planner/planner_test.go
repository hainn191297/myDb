package planner

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/sql/parser"
)

func TestBuildSelectPlan(t *testing.T) {
	ast := parser.AST{
		Type:       parser.SelectStmt,
		SchemaName: "public",
		TableName:  "users",
		Columns:    []string{"id", "name"},
		Where:      "id = 1",
	}

	// For now, SELECT requires a catalog (even if nil for basic plans)
	// This will be enhanced in later phases with semantic analysis
	plan, err := Build(context.Background(), ast, nil)
	if err == nil {
		t.Fatalf("Build SELECT should require catalog, but got plan: %v", plan)
	}
}

func TestBuildTxnOps(t *testing.T) {
	tests := []struct {
		stmt parser.StatementType
		act  TxnAction
	}{
		{parser.BeginStmt, TxnBegin},
		{parser.CommitStmt, TxnCommit},
		{parser.RollbackStmt, TxnRollback},
	}

	for _, tt := range tests {
		plan, err := Build(context.Background(), parser.AST{Type: tt.stmt}, nil) // TXN doesn't need catalog
		if err != nil {
			t.Fatalf("Build(%s): %v", tt.stmt, err)
		}
		op, ok := plan.Root.(*TxnOp)
		if !ok {
			t.Fatalf("expected TxnOp for %s, got %T", tt.stmt, plan.Root)
		}
		if op.Action != tt.act {
			t.Fatalf("txn action mismatch: got %s want %s", op.Action, tt.act)
		}
	}
}
