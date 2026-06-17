package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	dberrors "github.com/hainn191297/myDb/internal/errors"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

const (
	// SystemSchema is the schema name for system tables.
	SystemSchema = "__system"
	// CatalogTablesTable stores table definitions.
	CatalogTablesTable = "__catalog_tables"
	// CatalogColumnsTable stores column definitions.
	// CatalogIndexesTable stores index definitions.
	CatalogIndexesTable = "__catalog_indexes"
)

// Catalog manages schema metadata using system tables.
type Catalog struct {
	tableEngine engine.Engine // Engine for __catalog_tables
	indexEngine engine.Engine // Engine for __catalog_indexes
	mu          sync.RWMutex  // Protects cache
	cache       map[string]*TableDef
	indexCache  map[string][]IndexDef // table -> indexes
}

// TableDef represents a table's schema definition.
type TableDef struct {
	Schema  string      `json:"schema"`
	Table   string      `json:"table"`
	Columns []ColumnDef `json:"columns"`
}

// ColumnDef represents a column's schema definition.
type ColumnDef struct {
	Name         string   `json:"name"`
	Type         DataType `json:"type"`
	Nullable     bool     `json:"nullable"`
	PrimaryKey   bool     `json:"primary_key"`
	DefaultValue string   `json:"default_value,omitempty"`
}

// IndexDef represents an index definition.
type IndexDef struct {
	Schema       string   `json:"schema"`
	Table        string   `json:"table"`
	IndexName    string   `json:"index_name"`
	Columns      []string `json:"columns"`
	Unique       bool     `json:"unique"`
	IsPrimaryKey bool     `json:"is_primary_key"`
}

// NewCatalog creates a new catalog instance.
func NewCatalog(tableEngine, indexEngine engine.Engine) *Catalog {
	return &Catalog{
		tableEngine: tableEngine,
		indexEngine: indexEngine,
		cache:       make(map[string]*TableDef),
		indexCache:  make(map[string][]IndexDef),
	}
}

// LoadSystemTables initializes the catalog by loading all definitions
// from the system tables into the in-memory cache.
func (c *Catalog) LoadSystemTables(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Load Tables
	if err := c.loadTables(ctx); err != nil {
		return err
	}

	// Load Indexes
	if err := c.loadIndexes(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Catalog) loadTables(ctx context.Context) error {
	iter, err := c.tableEngine.Scan(ctx, nil, nil)
	if err != nil {
		if errors.Is(err, dberrors.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("catalog: scan tables: %w", err)
	}
	defer iter.Close()

	for iter.Next() {
		var tableDef TableDef
		if err := json.Unmarshal(iter.Value(), &tableDef); err != nil {
			return fmt.Errorf("catalog: unmarshal table def: %w", err)
		}
		key := c.tableKey(tableDef.Schema, tableDef.Table)
		c.cache[key] = &tableDef
	}
	return iter.Err()
}

func (c *Catalog) loadIndexes(ctx context.Context) error {
	iter, err := c.indexEngine.Scan(ctx, nil, nil)
	if err != nil {
		if errors.Is(err, dberrors.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("catalog: scan indexes: %w", err)
	}
	defer iter.Close()

	for iter.Next() {
		var indexDef IndexDef
		if err := json.Unmarshal(iter.Value(), &indexDef); err != nil {
			return fmt.Errorf("catalog: unmarshal index def: %w", err)
		}
		tableKey := c.tableKey(indexDef.Schema, indexDef.Table)
		c.indexCache[tableKey] = append(c.indexCache[tableKey], indexDef)
	}
	return iter.Err()
}

// CreateTable creates a new table definition in the catalog.
func (c *Catalog) CreateTable(ctx context.Context, schema, table string, columns []ColumnDef) error {
	if schema == "" || table == "" {
		return dberrors.ErrEmptyTableName
	}
	if len(columns) == 0 {
		return dberrors.ErrEmptyColumnList
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.tableKey(schema, table)
	if _, exists := c.cache[key]; exists {
		return fmt.Errorf("%w: %s.%s", dberrors.ErrTableExists, schema, table)
	}

	tableDef := &TableDef{
		Schema:  schema,
		Table:   table,
		Columns: columns,
	}

	// Serialize to JSON
	value, err := json.Marshal(tableDef)
	if err != nil {
		return fmt.Errorf("catalog: marshal table def: %w", err)
	}

	// Store in system table
	if err := c.tableEngine.Put(ctx, []byte(key), value); err != nil {
		return fmt.Errorf("catalog: store table def: %w", err)
	}

	// Update cache
	c.cache[key] = tableDef

	return nil
}

// DropTable removes a table definition from the catalog.
func (c *Catalog) DropTable(ctx context.Context, schema, table string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.tableKey(schema, table)
	if _, exists := c.cache[key]; !exists {
		return fmt.Errorf("%w: %s.%s", dberrors.ErrTableNotFound, schema, table)
	}

	// Delete from system table
	if err := c.tableEngine.Delete(ctx, []byte(key)); err != nil {
		return fmt.Errorf("catalog: delete table def: %w", err)
	}

	// Remove from cache
	delete(c.cache, key)
	// Also remove indexes for this table from cache (persistence handled by DropIndex calls usually, but for DropTable we should clean up)
	// In a real DB, DropTable drops all indexes.
	// For now, let's assume executor handles dropping indexes or we just clear cache.
	delete(c.indexCache, key)

	return nil
}

// GetTable retrieves a table definition from the catalog.
func (c *Catalog) GetTable(schema, table string) (*TableDef, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.tableKey(schema, table)
	tableDef, exists := c.cache[key]
	if !exists {
		return nil, fmt.Errorf("%w: %s.%s", dberrors.ErrTableNotFound, schema, table)
	}

	return tableDef, nil
}

// ListTables returns all table definitions in the catalog.
func (c *Catalog) ListTables() []*TableDef {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tables := make([]*TableDef, 0, len(c.cache))
	for _, tableDef := range c.cache {
		tables = append(tables, tableDef)
	}
	return tables
}

// CreateIndex creates a new index definition.
func (c *Catalog) CreateIndex(ctx context.Context, schema, table, indexName string, columns []string, unique, isPrimaryKey bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tableKey := c.tableKey(schema, table)
	if _, exists := c.cache[tableKey]; !exists {
		return fmt.Errorf("%w: %s.%s", dberrors.ErrTableNotFound, schema, table)
	}

	// Check if index already exists
	for _, idx := range c.indexCache[tableKey] {
		if idx.IndexName == indexName {
			return fmt.Errorf("%w: %s on %s.%s", dberrors.ErrIndexExists, indexName, schema, table)
		}
	}

	indexDef := IndexDef{
		Schema:       schema,
		Table:        table,
		IndexName:    indexName,
		Columns:      columns,
		Unique:       unique,
		IsPrimaryKey: isPrimaryKey,
	}

	value, err := json.Marshal(indexDef)
	if err != nil {
		return fmt.Errorf("catalog: marshal index def: %w", err)
	}

	indexKey := c.indexKey(schema, table, indexName)
	if err := c.indexEngine.Put(ctx, []byte(indexKey), value); err != nil {
		return fmt.Errorf("catalog: store index def: %w", err)
	}

	c.indexCache[tableKey] = append(c.indexCache[tableKey], indexDef)
	return nil
}

// DropIndex removes an index definition.
func (c *Catalog) DropIndex(ctx context.Context, schema, table, indexName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tableKey := c.tableKey(schema, table)
	indexes := c.indexCache[tableKey]

	idxPos := -1
	for i, idx := range indexes {
		if idx.IndexName == indexName {
			idxPos = i
			break
		}
	}

	if idxPos == -1 {
		return fmt.Errorf("%w: %s on %s.%s", dberrors.ErrIndexNotFound, indexName, schema, table)
	}

	indexKey := c.indexKey(schema, table, indexName)
	if err := c.indexEngine.Delete(ctx, []byte(indexKey)); err != nil {
		return fmt.Errorf("catalog: delete index def: %w", err)
	}

	// Remove from cache
	c.indexCache[tableKey] = append(indexes[:idxPos], indexes[idxPos+1:]...)
	return nil
}

// GetIndexes returns all indexes for a table.
func (c *Catalog) GetIndexes(schema, table string) ([]IndexDef, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tableKey := c.tableKey(schema, table)
	if _, exists := c.cache[tableKey]; !exists {
		return nil, fmt.Errorf("%w: %s.%s", dberrors.ErrTableNotFound, schema, table)
	}

	// Return a copy to prevent modification
	indexes := c.indexCache[tableKey]
	result := make([]IndexDef, len(indexes))
	copy(result, indexes)
	return result, nil
}

// FindIndexForColumn finds an index that starts with the given column.
// This is a simple heuristic for the optimizer.
func (c *Catalog) FindIndexForColumn(schema, table, column string) (*IndexDef, error) {
	indexes, err := c.GetIndexes(schema, table)
	if err != nil {
		return nil, err
	}

	for _, idx := range indexes {
		if len(idx.Columns) > 0 && idx.Columns[0] == column {
			return &idx, nil
		}
	}
	return nil, nil
}

// tableKey generates a unique key for a table (schema.table).
func (c *Catalog) tableKey(schema, table string) string {
	return schema + "." + table
}

// indexKey generates a unique key for an index (schema.table.indexName).
func (c *Catalog) indexKey(schema, table, indexName string) string {
	return schema + "." + table + "." + indexName
}
