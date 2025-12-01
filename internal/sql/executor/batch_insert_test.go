package executor

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/planner"
)

func TestExecutorBatchInsert(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider()

	// Setup fake engine for table
	eng := newFakeEngine()
	provider.engines["public.users"] = eng

	// Setup Catalog
	// We need a real catalog to resolve indexes, but we can mock the engine it uses?
	// Actually, executeInsert calls e.catalog.GetIndexes.
	// We can create a real catalog and populate it with system tables?
	// Or we can just mock the catalog if we could, but Catalog struct is concrete.
	//
	// Let's see if we can use a real catalog with fake engines.
	// Catalog uses tableEngine and indexEngine.
	// We can pass fake engines to NewCatalog.

	catTableEngine := newFakeEngine()
	catIndexEngine := newFakeEngine()
	catalog := schema.NewCatalog(catTableEngine, catIndexEngine)

	// Initialize system tables
	if err := catalog.LoadSystemTables(ctx); err != nil {
		// If system tables are empty, it might be fine, or we might need to init them.
		// LoadSystemTables tries to load from storage.
		// Since storage is empty, it should just init cache.
	}

	// Define table "users" in catalog
	err := catalog.CreateTable(ctx, "public", "users", []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
		{Name: "name", Type: schema.TypeText},
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Create Plan for Batch Insert
	// INSERT INTO users VALUES (1, 'alice'), (2, 'bob')
	// We construct the Plan manually to test Executor directly,
	// but we can also use Parser -> Planner if we want integration test.
	// Let's construct manually first to isolate Executor.

	// Encode values
	val1Id, _ := schema.TypeInt64.Encode("1")
	val1Name, _ := schema.TypeText.Encode("alice")
	val2Id, _ := schema.TypeInt64.Encode("2")
	val2Name, _ := schema.TypeText.Encode("bob")

	plan := planner.Plan{
		Root: &planner.InsertOp{
			Schema:  "public",
			Table:   "users",
			Columns: []string{"id", "name"},
			Values: [][][]byte{
				{val1Id, val1Name},
				{val2Id, val2Name},
			},
		},
	}

	exec := New(plan, Options{
		Provider: provider,
		Catalog:  catalog,
	})
	defer exec.Close()

	// Execute
	ok, err := exec.Next(ctx)
	if err != nil {
		t.Fatalf("exec.Next failed: %v", err)
	}
	if ok {
		t.Fatalf("expected no rows returned from INSERT")
	}

	// Verify data in engine
	// Row 1: key=1
	val1, err := eng.Get(ctx, val1Id)
	if err != nil {
		t.Fatalf("get row 1: %v", err)
	}
	// Verify content (length prefixed)
	// We don't have a decoder here easily, but we can check length at least.
	if len(val1) == 0 {
		t.Fatalf("row 1 empty")
	}

	// Row 2: key=2
	val2, err := eng.Get(ctx, val2Id)
	if err != nil {
		t.Fatalf("get row 2: %v", err)
	}
	if len(val2) == 0 {
		t.Fatalf("row 2 empty")
	}
}

func TestExecutorBatchInsertDuplicateKey(t *testing.T) {
	ctx := context.Background()
	provider := newFakeProvider()
	eng := newFakeEngine()
	provider.engines["public.users"] = eng

	catTableEngine := newFakeEngine()
	catIndexEngine := newFakeEngine()
	catalog := schema.NewCatalog(catTableEngine, catIndexEngine)
	catalog.CreateTable(ctx, "public", "users", []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
	})

	val1Id, _ := schema.TypeInt64.Encode("1")

	// Pre-insert key 1
	eng.Put(ctx, val1Id, []byte("existing"))

	plan := planner.Plan{
		Root: &planner.InsertOp{
			Schema:  "public",
			Table:   "users",
			Columns: []string{"id"},
			Values: [][][]byte{
				{val1Id}, // Duplicate
			},
		},
	}

	exec := New(plan, Options{
		Provider: provider,
		Catalog:  catalog,
	})
	defer exec.Close()

	_, err := exec.Next(ctx)
	if err == nil {
		t.Fatalf("expected duplicate key error")
	}
}

func BenchmarkBatchInsert(b *testing.B) {
	ctx := context.Background()
	provider := newFakeProvider()
	eng := newFakeEngine()
	provider.engines["public.users"] = eng

	catTableEngine := newFakeEngine()
	catIndexEngine := newFakeEngine()
	catalog := schema.NewCatalog(catTableEngine, catIndexEngine)
	catalog.CreateTable(ctx, "public", "users", []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
		{Name: "name", Type: schema.TypeText},
	})

	// Prepare batch values
	batchSize := 1000
	rows := make([][][]byte, batchSize)
	for i := 0; i < batchSize; i++ {
		id, _ := schema.TypeInt64.Encode(i)
		name, _ := schema.TypeText.Encode("benchmark")
		rows[i] = [][]byte{id, name}
	}

	plan := planner.Plan{
		Root: &planner.InsertOp{
			Schema:  "public",
			Table:   "users",
			Columns: []string{"id", "name"},
			Values:  rows,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We need to reset engine or use different keys?
		// For simplicity, we just overwrite (fake engine supports overwrite)
		exec := New(plan, Options{Catalog: catalog, Provider: provider})
		exec.Next(ctx)
	}
}

func BenchmarkSingleInsert(b *testing.B) {
	ctx := context.Background()
	provider := newFakeProvider()
	eng := newFakeEngine()
	provider.engines["public.users"] = eng

	catTableEngine := newFakeEngine()
	catIndexEngine := newFakeEngine()
	catalog := schema.NewCatalog(catTableEngine, catIndexEngine)
	catalog.CreateTable(ctx, "public", "users", []schema.ColumnDef{
		{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
		{Name: "name", Type: schema.TypeText},
	})

	// Prepare single values
	batchSize := 1000
	rows := make([][][][]byte, batchSize)
	for i := 0; i < batchSize; i++ {
		id, _ := schema.TypeInt64.Encode(i)
		name, _ := schema.TypeText.Encode("benchmark")
		rows[i] = [][][]byte{{id, name}}
	}

	plans := make([]planner.Plan, batchSize)
	for i := 0; i < batchSize; i++ {
		plans[i] = planner.Plan{
			Root: &planner.InsertOp{
				Schema:  "public",
				Table:   "users",
				Columns: []string{"id", "name"},
				Values:  rows[i],
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 1000 single inserts
		for j := 0; j < batchSize; j++ {
			exec := New(plans[j], Options{Catalog: catalog, Provider: provider})
			exec.Next(ctx)
		}
	}
}
