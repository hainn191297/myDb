package planner

import (
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

	plan, err := Build(ast, nil) // SELECT doesn't need catalog
	if err != nil {
		t.Fatalf("Build SELECT: %v", err)
	}

	scan, ok := plan.Root.(*SeqScanOp)
	if !ok {
		t.Fatalf("expected SeqScanOp, got %T", plan.Root)
	}
	if scan.Schema != "public" || scan.Table != "users" {
		t.Fatalf("schema/table mismatch: %s.%s", scan.Schema, scan.Table)
	}
	if len(scan.Columns) != 2 {
		t.Fatalf("columns size mismatch: %v", scan.Columns)
	}
	if scan.Filter != "id = 1" {
		t.Fatalf("filter mismatch: %s", scan.Filter)
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
		plan, err := Build(parser.AST{Type: tt.stmt}, nil) // TXN doesn't need catalog
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
