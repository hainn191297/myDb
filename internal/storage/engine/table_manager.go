package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
TableManager encapsulates filesystem layout rules for schemas/tables.

Responsibilities:
- ensure schema directories exist under basePath
- map each logical table to a single data file <schema>/<table>.db
- cache open FileManagers for reuse
*/
type TableManager struct {
	basePath string // e.g., "data/"

	mu     sync.Mutex
	tables map[string]*page.FileManager
}

func NewTableManager(basePath string) *TableManager {
	return &TableManager{
		basePath: basePath,
		tables:   make(map[string]*page.FileManager),
	}
}

/*
CreateSchema ensures the schema directory exists (idempotent).
*/
func (tm *TableManager) CreateSchema(schema string) error {
	return os.MkdirAll(filepath.Join(tm.basePath, schema), 0755)
}

/*
OpenTable lazily creates (if needed) and caches a FileManager bound to
data/<schema>/<table>.db so higher layers can read/write pages.
*/
func (tm *TableManager) OpenTable(schema, table string) (*page.FileManager, error) {
	key := tm.tableKey(schema, table)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if fm, ok := tm.tables[key]; ok {
		return fm, nil
	}

	dir := filepath.Join(tm.basePath, schema)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file := filepath.Join(dir, table+".db")
	fm, err := page.NewFileManager(file)
	if err != nil {
		return nil, err
	}

	tm.tables[key] = fm
	return fm, nil
}

/*
CreateTable is kept for backward compatibility and proxies to OpenTable.
*/
func (tm *TableManager) CreateTable(schema, table string) (*page.FileManager, error) {
	return tm.OpenTable(schema, table)
}

/*
Close releases all cached FileManagers.
*/
func (tm *TableManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errs []error
	for key, fm := range tm.tables {
		if err := fm.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
		delete(tm.tables, key)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (tm *TableManager) tableKey(schema, table string) string {
	return schema + "." + table
}
