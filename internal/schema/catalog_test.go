package schema

import (
	"context"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/engine"
)

func TestCatalogCreateTable(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := NewCatalog(tableEng, indexEng)

	columns := []ColumnDef{
		{Name: "id", Type: TypeInt64, Nullable: false},
		{Name: "name", Type: TypeText, Nullable: true},
	}

	err := catalog.CreateTable(ctx, "public", "users", columns)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// Verify in cache
	tableDef, err := catalog.GetTable("public", "users")
	if err != nil {
		t.Fatalf("GetTable failed: %v", err)
	}
	if tableDef.Schema != "public" || tableDef.Table != "users" {
		t.Errorf("table mismatch: got %s.%s", tableDef.Schema, tableDef.Table)
	}
	if len(tableDef.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(tableDef.Columns))
	}
}

func TestCatalogCreateTableDuplicate(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := NewCatalog(tableEng, indexEng)

	columns := []ColumnDef{
		{Name: "id", Type: TypeInt64},
	}

	_ = catalog.CreateTable(ctx, "public", "users", columns)
	err := catalog.CreateTable(ctx, "public", "users", columns)
	if err == nil {
		t.Error("expected error for duplicate table, got nil")
	}
}

func TestCatalogDropTable(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := NewCatalog(tableEng, indexEng)

	columns := []ColumnDef{
		{Name: "id", Type: TypeInt64},
	}

	_ = catalog.CreateTable(ctx, "public", "users", columns)

	err := catalog.DropTable(ctx, "public", "users")
	if err != nil {
		t.Fatalf("DropTable failed: %v", err)
	}

	// Verify removed
	_, err = catalog.GetTable("public", "users")
	if err == nil {
		t.Error("expected error for dropped table, got nil")
	}
}

func TestCatalogPersistence(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog1 := NewCatalog(tableEng, indexEng)

	columns := []ColumnDef{
		{Name: "id", Type: TypeInt64},
		{Name: "email", Type: TypeText},
	}

	_ = catalog1.CreateTable(ctx, "public", "users", columns)

	// Create a new catalog instance with same engine
	catalog2 := NewCatalog(tableEng, indexEng)
	if err := catalog2.LoadSystemTables(ctx); err != nil {
		t.Fatalf("LoadSystemTables failed: %v", err)
	}

	// Verify table exists in new catalog
	tableDef, err := catalog2.GetTable("public", "users")
	if err != nil {
		t.Fatalf("GetTable in new catalog failed: %v", err)
	}
	if len(tableDef.Columns) != 2 {
		t.Errorf("expected 2 columns in reloaded catalog, got %d", len(tableDef.Columns))
	}
}

func TestCatalogListTables(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := NewCatalog(tableEng, indexEng)

	_ = catalog.CreateTable(ctx, "public", "users", []ColumnDef{{Name: "id", Type: TypeInt64}})
	_ = catalog.CreateTable(ctx, "public", "orders", []ColumnDef{{Name: "id", Type: TypeInt64}})

	tables := catalog.ListTables()
	if len(tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(tables))
	}
}

func TestCatalogIndexOperations(t *testing.T) {
	ctx := context.Background()
	tableEng := newFakeEngine()
	indexEng := newFakeEngine()
	catalog := NewCatalog(tableEng, indexEng)

	// Create table first
	_ = catalog.CreateTable(ctx, "public", "users", []ColumnDef{
		{Name: "id", Type: TypeInt64},
		{Name: "email", Type: TypeText},
	})

	// Create Index
	err := catalog.CreateIndex(ctx, "public", "users", "idx_email", []string{"email"}, true, false)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify GetIndexes
	indexes, err := catalog.GetIndexes("public", "users")
	if err != nil {
		t.Fatalf("GetIndexes failed: %v", err)
	}
	if len(indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(indexes))
	}
	if indexes[0].IndexName != "idx_email" || !indexes[0].Unique {
		t.Errorf("index mismatch: %+v", indexes[0])
	}

	// Verify FindIndexForColumn
	idx, err := catalog.FindIndexForColumn("public", "users", "email")
	if err != nil {
		t.Fatalf("FindIndexForColumn failed: %v", err)
	}
	if idx == nil {
		t.Error("expected to find index for email column")
	} else if idx.IndexName != "idx_email" {
		t.Errorf("expected idx_email, got %s", idx.IndexName)
	}

	// Drop Index
	err = catalog.DropIndex(ctx, "public", "users", "idx_email")
	if err != nil {
		t.Fatalf("DropIndex failed: %v", err)
	}

	// Verify removed
	indexes, _ = catalog.GetIndexes("public", "users")
	if len(indexes) != 0 {
		t.Errorf("expected 0 indexes after drop, got %d", len(indexes))
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
	var pairs []kvPair
	for k, v := range f.data {
		pairs = append(pairs, kvPair{key: []byte(k), value: v})
	}
	return &fakeIterator{pairs: pairs}, nil
}

type kvPair struct {
	key   []byte
	value []byte
}

type fakeIterator struct {
	pairs []kvPair
	idx   int
}

func (it *fakeIterator) Next() bool {
	if it.idx >= len(it.pairs) {
		return false
	}
	it.idx++
	return true
}

func (it *fakeIterator) Key() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}
	return it.pairs[it.idx-1].key
}

func (it *fakeIterator) Value() []byte {
	if it.idx == 0 || it.idx > len(it.pairs) {
		return nil
	}
	return it.pairs[it.idx-1].value
}

func (it *fakeIterator) Err() error {
	return nil
}

func (it *fakeIterator) Close() error {
	return nil
}
