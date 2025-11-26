package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

func TestExecutorCreateTable(t *testing.T) {
	ctx := context.Background()
	catalog := setupTestCatalog(t)

	// Parse and plan
	sql := "CREATE TABLE users (id INT, name TEXT, active BOOL)"
	ast, err := parser.Parse(context.Background(), sql)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	plan, err := planner.Build(context.Background(), ast, catalog)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Execute
	exec := New(plan, Options{Catalog: catalog})
	_, err = exec.Next(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify table exists in catalog
	tableDef, err := catalog.GetTable("public", "users")
	if err != nil {
		t.Fatalf("GetTable failed: %v", err)
	}
	if len(tableDef.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(tableDef.Columns))
	}
	if tableDef.Columns[0].Name != "id" || tableDef.Columns[0].Type != schema.TypeInt64 {
		t.Errorf("column 0 mismatch: %+v", tableDef.Columns[0])
	}
}

func TestExecutorCreateTableDuplicate(t *testing.T) {
	ctx := context.Background()
	catalog := setupTestCatalog(t)

	sql := "CREATE TABLE users (id INT)"
	ast, _ := parser.Parse(context.Background(), sql)
	plan, _ := planner.Build(context.Background(), ast, catalog)

	// First create should succeed
	exec := New(plan, Options{Catalog: catalog})
	_, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("First CREATE TABLE failed: %v", err)
	}

	// Second create should fail
	exec = New(plan, Options{Catalog: catalog})
	_, err = exec.Next(ctx)
	if err == nil {
		t.Error("expected error for duplicate table, got nil")
	}
}

func TestExecutorDropTable(t *testing.T) {
	ctx := context.Background()
	catalog := setupTestCatalog(t)

	// Create table first
	createSQL := "CREATE TABLE users (id INT)"
	createAST, _ := parser.Parse(context.Background(), createSQL)
	createPlan, _ := planner.Build(context.Background(), createAST, catalog)
	exec := New(createPlan, Options{Catalog: catalog})
	_, _ = exec.Next(ctx)

	// Drop table
	dropSQL := "DROP TABLE users"
	dropAST, err := parser.Parse(context.Background(), dropSQL)
	if err != nil {
		t.Fatalf("Parse DROP failed: %v", err)
	}

	dropPlan, err := planner.Build(context.Background(), dropAST, catalog)
	if err != nil {
		t.Fatalf("Build DROP failed: %v", err)
	}

	exec = New(dropPlan, Options{Catalog: catalog})
	_, err = exec.Next(ctx)
	if err != nil {
		t.Fatalf("Execute DROP failed: %v", err)
	}

	// Verify table no longer exists
	_, err = catalog.GetTable("public", "users")
	if err == nil {
		t.Error("expected error for dropped table, got nil")
	}
}

func TestExecutorDDLEndToEnd(t *testing.T) {
	ctx := context.Background()
	catalog := setupTestCatalog(t)

	// CREATE TABLE
	_, err := executeDDL(ctx, catalog, "CREATE TABLE users (id INT, email TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// Verify created
	tableDef, err := catalog.GetTable("public", "users")
	if err != nil {
		t.Fatalf("GetTable failed: %v", err)
	}
	if len(tableDef.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(tableDef.Columns))
	}

	// DROP TABLE
	_, err = executeDDL(ctx, catalog, "DROP TABLE users")
	if err != nil {
		t.Fatalf("DROP TABLE failed: %v", err)
	}

	// Verify dropped
	_, err = catalog.GetTable("public", "users")
	if err == nil {
		t.Error("table should not exist after DROP")
	}
}

func setupTestCatalog(t *testing.T) *schema.Catalog {
	t.Helper()
	tableEng := newFakeEngineForCatalog()
	indexEng := newFakeEngineForCatalog()
	catalog := schema.NewCatalog(tableEng, indexEng)
	return catalog
}

func executeDDL(ctx context.Context, catalog *schema.Catalog, sql string) (bool, error) {
	ast, err := parser.Parse(context.Background(), sql)
	if err != nil {
		return false, err
	}

	plan, err := planner.Build(context.Background(), ast, catalog)
	if err != nil {
		return false, err
	}

	exec := New(plan, Options{Catalog: catalog})
	return exec.Next(ctx)
}

// Fake engine for catalog testing
type fakeEngineForCatalog struct {
	data map[string][]byte
}

func newFakeEngineForCatalog() *fakeEngineForCatalog {
	return &fakeEngineForCatalog{data: make(map[string][]byte)}
}

func (f *fakeEngineForCatalog) Get(ctx context.Context, key []byte) ([]byte, error) {
	if val, ok := f.data[string(key)]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (f *fakeEngineForCatalog) Put(ctx context.Context, key, value []byte) error {
	f.data[string(key)] = value
	return nil
}

func (f *fakeEngineForCatalog) Delete(ctx context.Context, key []byte) error {
	delete(f.data, string(key))
	return nil
}

func (f *fakeEngineForCatalog) Scan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	var pairs []kvPairForCatalog
	for k, v := range f.data {
		pairs = append(pairs, kvPairForCatalog{key: []byte(k), value: v})
	}
	return &fakeIteratorForCatalog{pairs: pairs}, nil
}

type kvPairForCatalog struct {
	key   []byte
	value []byte
}

type fakeIteratorForCatalog struct {
	pairs []kvPairForCatalog
	idx   int
}

func (it *fakeIteratorForCatalog) Next() bool {
	if it.idx >= len(it.pairs) {
		return false
	}
	it.idx++
	return true
}

func (it *fakeIteratorForCatalog) Key() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}
	return it.pairs[it.idx-1].key
}

func (it *fakeIteratorForCatalog) Value() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}
	return it.pairs[it.idx-1].value
}

func (it *fakeIteratorForCatalog) Err() error {
	return nil
}

func (it *fakeIteratorForCatalog) Close() error {
	return nil
}
