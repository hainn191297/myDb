package executor

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

func TestExecutorInsert(t *testing.T) {
	ctx := context.Background()
	catalog, provider := setupDMLTest(t)

	// Create table
	executeDDL(ctx, catalog, "CREATE TABLE users (id INT, name TEXT)")
	setupTableEngine(provider, "public", "users")

	// Parse and execute INSERT
	sql := "INSERT INTO users VALUES (1, 'alice')"
	ast, err := parser.Parse(sql)
	if err != nil {
		t.Fatalf("Parse INSERT: %v", err)
	}

	plan, err := planner.Build(ast, catalog)
	if err != nil {
		t.Fatalf("Build INSERT: %v", err)
	}

	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	_, err = exec.Next(ctx)
	if err != nil {
		t.Fatalf("Execute INSERT: %v", err)
	}

	// Verify row was inserted by scanning
	eng, _ := provider.Engine("public", "users")
	iter, _ := eng.Scan(ctx, nil, nil)
	defer iter.Close()

	if !iter.Next() {
		t.Fatal("expected at least one row after INSERT")
	}
}

func TestExecutorUpdate(t *testing.T) {
	ctx := context.Background()
	catalog, provider := setupDMLTest(t)

	// Create table and insert data
	executeDDL(ctx, catalog, "CREATE TABLE users (id INT, name TEXT)")
	setupTableEngine(provider, "public", "users")
	executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (1, 'alice')")

	// Update
	sql := "UPDATE users SET name = 'bob'"
	ast, _ := parser.Parse(sql)
	plan, _ := planner.Build(ast, catalog)

	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	_, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("Execute UPDATE: %v", err)
	}

	// Verify update (basic check - row still exists)
	eng, _ := provider.Engine("public", "users")
	iter, _ := eng.Scan(ctx, nil, nil)
	defer iter.Close()

	if !iter.Next() {
		t.Fatal("expected row after UPDATE")
	}
}

func TestExecutorDelete(t *testing.T) {
	ctx := context.Background()
	catalog, provider := setupDMLTest(t)

	// Create table and insert data
	executeDDL(ctx, catalog, "CREATE TABLE users (id INT, name TEXT)")
	setupTableEngine(provider, "public", "users")
	executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (1, 'alice')")

	// Delete all
	sql := "DELETE FROM users"
	ast, _ := parser.Parse(sql)
	plan, _ := planner.Build(ast, catalog)

	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	_, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("Execute DELETE: %v", err)
	}

	// Verify deletion
	eng, _ := provider.Engine("public", "users")
	iter, _ := eng.Scan(ctx, nil, nil)
	defer iter.Close()

	if iter.Next() {
		t.Error("expected no rows after DELETE")
	}
}

func TestCRUDWorkflow(t *testing.T) {
	ctx := context.Background()
	catalog, provider := setupDMLTest(t)

	// CREATE TABLE
	executeDDL(ctx, catalog, "CREATE TABLE users (id INT, email TEXT, active BOOL)")
	setupTableEngine(provider, "public", "users") // Setup engine after CREATE TABLE

	// Verify table in catalog
	tableDef, err := catalog.GetTable("public", "users")
	if err != nil {
		t.Fatalf("Table not in catalog: %v", err)
	}
	if len(tableDef.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(tableDef.Columns))
	}

	// INSERT row 1
	err = executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (1, 'alice@ex.com', true)")
	if err != nil {
		t.Fatalf("INSERT 1 failed: %v", err)
	}

	// INSERT row 2
	err = executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (2, 'bob@ex.com', false)")
	if err != nil {
		t.Fatalf("INSERT 2 failed: %v", err)
	}

	// Verify 2 rows exist
	eng, _ := provider.Engine("public", "users")
	rowCount := countRows(ctx, t, eng)
	if rowCount != 2 {
		t.Errorf("expected 2 rows after INSERTs, got %d", rowCount)
	}

	// UPDATE row 2
	err = executeDML(ctx, catalog, provider, "UPDATE users SET active = true")
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}

	// DELETE row 1
	err = executeDML(ctx, catalog, provider, "DELETE FROM users")
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	// Verify all rows deleted
	rowCount = countRows(ctx, t, eng)
	if rowCount != 0 {
		t.Errorf("expected 0 rows after DELETE, got %d", rowCount)
	}

	// DROP TABLE
	executeDDL(ctx, catalog, "DROP TABLE users")

	// Verify table dropped
	_, err = catalog.GetTable("public", "users")
	if err == nil {
		t.Error("table should not exist after DROP")
	}
}

func setupDMLTest(t *testing.T) (*schema.Catalog, *fakeProvider) {
	t.Helper()
	tableEng := newFakeEngineForCatalog()
	indexEng := newFakeEngineForCatalog()
	catalog := schema.NewCatalog(tableEng, indexEng)

	// Pre-create engines for tables (fakeProvider needs them)
	provider := &fakeProvider{engines: make(map[string]engine.Engine)}

	return catalog, provider
}

// Helper to setup engine for a table after creation
func setupTableEngine(provider *fakeProvider, schema, table string) {
	eng := newFakeEngineForCatalog()
	provider.engines[schema+"."+table] = eng
}

func executeDML(ctx context.Context, catalog *schema.Catalog, provider EngineProvider, sql string) error {
	ast, err := parser.Parse(sql)
	if err != nil {
		return err
	}

	plan, err := planner.Build(ast, catalog)
	if err != nil {
		return err
	}

	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	_, err = exec.Next(ctx)
	return err
}

func countRows(ctx context.Context, t *testing.T, eng engine.Engine) int {
	t.Helper()
	iter, err := eng.Scan(ctx, nil, nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		count++
	}
	return count
}
