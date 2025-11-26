package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/executor"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

// End-to-end recovery test: insert via executor, delete data files, run WAL recovery, ensure data visible.
func TestProviderRecoveryRestoresData(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	prov, err := New(dir, 8)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	cat, err := prov.LoadCatalog(ctx)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	// create table (no index to force seq scan)
	cols := []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64},
		{Name: "name", Type: schema.TypeText},
	}
	if err := cat.CreateTable(ctx, "public", "users", cols); err != nil {
		t.Fatalf("create table: %v", err)
	}

	// insert a row
	insertSQL := "INSERT INTO users VALUES (1, 'alice')"
	ast, err := parser.Parse(context.Background(), insertSQL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	plan, err := planner.Build(context.Background(), ast, cat)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	exec := executor.New(plan, executor.Options{
		TxnManager: nil,
		SessionTxn: nil,
		Provider:   prov,
		Catalog:    cat,
	})
	if _, err := exec.Next(ctx); err != nil {
		t.Fatalf("execute insert: %v", err)
	}
	exec.Close()

	// flush and close to simulate clean shutdown of handles
	if err := prov.Close(); err != nil {
		t.Fatalf("close provider: %v", err)
	}

	// simulate data loss: remove data file after WAL persisted
	dataPath := filepath.Join(dir, "public", "users.db")
	if err := os.Remove(dataPath); err != nil {
		t.Fatalf("remove data file: %v", err)
	}
	walPath := filepath.Join(dir, "public", "users.wal")
	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("wal file missing: %v", err)
	}

	// recover from WAL using fresh manager
	wm := wal.NewManager(dir)
	if err := wm.Recover(); err != nil {
		t.Fatalf("wal recover: %v", err)
	}

	// open fresh provider/catalog and read row
	prov2, err := New(dir, 8)
	if err != nil {
		t.Fatalf("new provider 2: %v", err)
	}
	defer prov2.Close()
	cat2, err := prov2.LoadCatalog(ctx)
	if err != nil {
		t.Fatalf("load catalog 2: %v", err)
	}

	selectSQL := "SELECT * FROM users WHERE id = 1"
	ast, err = parser.Parse(context.Background(), selectSQL)
	if err != nil {
		t.Fatalf("parse select: %v", err)
	}
	plan, err = planner.Build(context.Background(), ast, cat2)
	if err != nil {
		t.Fatalf("plan select: %v", err)
	}
	exec = executor.New(plan, executor.Options{
		Provider: prov2,
		Catalog:  cat2,
	})
	defer exec.Close()
	ok, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("select next: %v", err)
	}
	if !ok {
		t.Fatalf("expected row after recovery")
	}
	row := exec.Row()
	if len(row.Values) < 2 || string(row.Values[1]) != "alice" {
		t.Fatalf("unexpected row %+v", row)
	}
}
