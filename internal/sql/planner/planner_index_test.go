package planner

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

func TestPlanCreateIndex(t *testing.T) {
	sql := "CREATE INDEX idx_email ON users (email)"
	ast, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	plan, err := Build(ast, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	op, ok := plan.Root.(*CreateIndexOp)
	if !ok {
		t.Fatalf("expected CreateIndexOp, got %T", plan.Root)
	}

	if op.IndexName != "idx_email" {
		t.Errorf("expected index name idx_email, got %s", op.IndexName)
	}
	if op.Table != "users" {
		t.Errorf("expected table users, got %s", op.Table)
	}
	if len(op.Columns) != 1 || op.Columns[0] != "email" {
		t.Errorf("expected column email, got %v", op.Columns)
	}
	if op.Unique {
		t.Error("expected not unique")
	}
}

func TestPlanDropIndex(t *testing.T) {
	sql := "DROP INDEX idx_email ON users"
	ast, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	plan, err := Build(ast, nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	op, ok := plan.Root.(*DropIndexOp)
	if !ok {
		t.Fatalf("expected DropIndexOp, got %T", plan.Root)
	}

	if op.IndexName != "idx_email" {
		t.Errorf("expected index name idx_email, got %s", op.IndexName)
	}
	if op.Table != "users" {
		t.Errorf("expected table users, got %s", op.Table)
	}
}

func TestPlanIndexScanSelection(t *testing.T) {
	// Setup catalog with an index
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := schema.NewCatalog(tableEng, indexEng)
	ctx := context.Background()

	_ = catalog.CreateTable(ctx, "public", "users", []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64},
		{Name: "email", Type: schema.TypeText},
	})
	_ = catalog.CreateIndex(ctx, "public", "users", "idx_email", []string{"email"}, false, false)

	tests := []struct {
		name     string
		sql      string
		wantType string // "SeqScan" or "IndexScan"
	}{
		{
			name:     "No WHERE clause",
			sql:      "SELECT * FROM users",
			wantType: "SeqScan",
		},
		{
			name:     "WHERE on non-indexed column",
			sql:      "SELECT * FROM users WHERE id = 1",
			wantType: "SeqScan",
		},
		{
			name:     "WHERE on indexed column",
			sql:      "SELECT * FROM users WHERE email = 'alice@example.com'",
			wantType: "IndexScan",
		},
		{
			name: "WHERE complex (no simple =)",
			// "id > 5" is not a simple equality check, so should be SeqScan
			sql:      "SELECT * FROM users WHERE id > 5",
			wantType: "SeqScan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			plan, err := Build(ast, catalog)
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			switch tt.wantType {
			case "SeqScan":
				if _, ok := plan.Root.(*SeqScanOp); !ok {
					t.Errorf("expected SeqScanOp, got %T", plan.Root)
				}
			case "IndexScan":
				if _, ok := plan.Root.(*IndexScanOp); !ok {
					t.Errorf("expected IndexScanOp, got %T", plan.Root)
				}
			}
		})
	}
}

// Fake engine for testing
type fakeEngine struct {
	data map[string][]byte
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{data: make(map[string][]byte)}
}

func (f *fakeEngine) Get(ctx context.Context, key []byte) ([]byte, error) {
	if val, ok := f.data[string(key)]; ok {
		return val, nil
	}
	return nil, engine.ErrKeyNotFound
}

func (f *fakeEngine) Put(ctx context.Context, key, value []byte) error {
	f.data[string(key)] = value
	return nil
}

func (f *fakeEngine) Delete(ctx context.Context, key []byte) error {
	delete(f.data, string(key))
	return nil
}

func (f *fakeEngine) Scan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	return &fakeIterator{}, nil
}

type fakeIterator struct{}

func (it *fakeIterator) Next() bool    { return false }
func (it *fakeIterator) Key() []byte   { return nil }
func (it *fakeIterator) Value() []byte { return nil }
func (it *fakeIterator) Err() error    { return nil }
func (it *fakeIterator) Close() error  { return nil }
