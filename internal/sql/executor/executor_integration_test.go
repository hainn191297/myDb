package executor

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/engine/btree"
)

// integrationProvider extends fakeProvider to support MemIndexEngine
type integrationProvider struct {
	*fakeProvider
}

func newIntegrationProvider() *integrationProvider {
	return &integrationProvider{
		fakeProvider: newFakeProvider(),
	}
}

// Index overrides fakeProvider.Index to return a real in-memory B+ Tree engine
func (p *integrationProvider) Index(schema, table, indexName string) (engine.IndexEngine, error) {
	key := schema + "." + table + "." + indexName
	if idx, ok := p.indexes[key]; ok {
		return idx, nil
	}
	// Create new MemIndexEngine
	idx := btree.NewMemEngine()
	p.indexes[key] = idx
	return idx, nil
}

func TestExecutorIntegration_EndToEnd(t *testing.T) {
	ctx := context.Background()

	// Setup
	provider := newIntegrationProvider()
	catalog := schema.NewCatalog(newFakeEngineForCatalog(), newFakeEngineForCatalog())

	// 1. CREATE TABLE
	sql := "CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT)"
	executeDDLWithProvider(ctx, catalog, provider, sql)
	// Provision table storage after metadata exists
	setupTableEngine(provider.fakeProvider, "public", "users")

	// 2. INSERT Data
	executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (1, 'Alice', 30)")
	executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (2, 'Bob', 25)")
	executeDML(ctx, catalog, provider, "INSERT INTO users VALUES (3, 'Charlie', 35)")

	// 3. CREATE INDEX on age
	executeDDLWithProvider(ctx, catalog, provider, "CREATE INDEX idx_age ON users (age)")

	// 4. SELECT with Index Scan (age = 25)
	// We need to ensure planner picks IndexScan.
	sql = "SELECT * FROM users WHERE age = 25"
	ast, _ := parser.Parse(context.Background(), sql)
	plan, _ := planner.Build(context.Background(), ast, catalog)

	if _, ok := plan.Root.(*planner.IndexScanOp); !ok {
		t.Fatalf("Expected IndexScanOp for query on indexed column 'age', got %T", plan.Root)
	}

	exec := New(plan, Options{Catalog: catalog, Provider: provider})
	ok, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if !ok {
		t.Fatal("Expected result for age=25")
	}
	row := exec.Row()
	if string(row.Values[1]) != "Bob" {
		t.Errorf("Expected Bob, got %s", row.Values[1])
	}
	if ok, _ := exec.Next(ctx); ok { // Should be only one result
		t.Error("Expected only one result")
	}

	// 5. UPDATE (Update indexed column)
	// Update Bob's age to 26
	executeDML(ctx, catalog, provider, "UPDATE users SET age = 26 WHERE id = 2")

	// Verify old index entry is gone
	sql = "SELECT * FROM users WHERE age = 25"
	ast, _ = parser.Parse(context.Background(), sql)
	plan, _ = planner.Build(context.Background(), ast, catalog)
	exec = New(plan, Options{Catalog: catalog, Provider: provider})
	ok, _ = exec.Next(ctx)
	if ok {
		t.Error("Expected no result for age=25 after update")
	}

	// Verify new index entry exists
	sql = "SELECT * FROM users WHERE age = 26"
	ast, _ = parser.Parse(context.Background(), sql)
	plan, _ = planner.Build(context.Background(), ast, catalog)
	exec = New(plan, Options{Catalog: catalog, Provider: provider})
	ok, _ = exec.Next(ctx)
	if !ok {
		t.Error("Expected result for age=26 after update")
	} else {
		row = exec.Row()
		if string(row.Values[1]) != "Bob" {
			t.Errorf("Expected Bob, got %s", row.Values[1])
		}
	}

	// 6. DELETE
	executeDML(ctx, catalog, provider, "DELETE FROM users WHERE id = 2")

	// Verify deleted from index
	sql = "SELECT * FROM users WHERE age = 26"
	ast, _ = parser.Parse(context.Background(), sql)
	plan, _ = planner.Build(context.Background(), ast, catalog)
	exec = New(plan, Options{Catalog: catalog, Provider: provider})
	ok, _ = exec.Next(ctx)
	if ok {
		t.Error("Expected no result after delete")
	}

	// Verify deleted from PK index (lookup by ID)
	sql = "SELECT * FROM users WHERE id = 2"
	// Note: Planner might choose SeqScan or IndexScan for PK depending on implementation.
	// Currently our heuristic prefers IndexScan if available.
	// PK creates an index named __pk_users_id
	ast, _ = parser.Parse(context.Background(), sql)
	plan, _ = planner.Build(context.Background(), ast, catalog)
	exec = New(plan, Options{Catalog: catalog, Provider: provider})
	ok, _ = exec.Next(ctx)
	if ok {
		t.Error("Expected no result for id=2 after delete")
	}
}
