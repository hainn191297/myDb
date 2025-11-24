package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/storage/buffer"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/engine/btree"
	"github.com/hainn191297/myDb/internal/storage/engine/heap"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

// Provider wires together table/index engines with shared buffer/table managers.
// It implements executor.EngineProvider so the SQL layer can resolve engines by
// schema/table/index name without knowing storage internals.
type Provider struct {
	tm *engine.TableManager
	bm *engine.BufferManager
	wm *wal.Manager

	mu           sync.Mutex
	tableEngines map[string]engine.Engine
	indexEngines map[string]engine.IndexEngine

	// Catalog for accessing table schema (can be nil initially)
	catalog *schema.Catalog
}

// New creates a storage provider backed by heap tables and an in-memory B+Tree
// index. WAL is not yet connected; durability is handled by page writes.
func New(basePath string, poolPages int) (*Provider, error) {
	if basePath == "" {
		basePath = "data"
	}
	if poolPages <= 0 {
		poolPages = 16
	}

	walMgr := wal.NewManager(basePath)
	if err := walMgr.Recover(); err != nil {
		return nil, fmt.Errorf("provider: wal recovery: %w", err)
	}

	tm := engine.NewTableManager(basePath)
	dispatcher := newWalDispatcher(tm, walMgr)
	pool := buffer.NewGlobalPool(poolPages, dispatcher)
	bm := engine.NewBufferManager(tm, pool, dispatcher)

	return &Provider{
		tm:           tm,
		bm:           bm,
		wm:           walMgr,
		tableEngines: make(map[string]engine.Engine),
		indexEngines: make(map[string]engine.IndexEngine),
	}, nil
}

// Engine returns (and caches) a heap engine for the given table.
func (p *Provider) Engine(schemaName, table string) (engine.Engine, error) {
	if schemaName == "" || table == "" {
		return nil, fmt.Errorf("provider: schema and table required")
	}
	key := schemaName + "." + table

	p.mu.Lock()
	defer p.mu.Unlock()

	if eng, ok := p.tableEngines[key]; ok {
		return eng, nil
	}

	// Try to get TableDef from catalog (may be nil if catalog not set yet)
	var tableDef *schema.TableDef
	if p.catalog != nil {
		if td, err := p.catalog.GetTable(schemaName, table); err == nil {
			tableDef = td
		}
		// If error, just pass nil and fall back to O(n) scans
	}

	eng := heap.NewHeapEngine(schemaName, table, p.tm, p.bm, tableDef, p)
	p.tableEngines[key] = eng
	return eng, nil
}

// Index returns (and caches) an index engine for the given table/index name.
// Currently uses the in-memory B+Tree; on-disk implementation will replace it.
func (p *Provider) Index(schemaName, table, indexName string) (engine.IndexEngine, error) {
	if schemaName == "" || table == "" || indexName == "" {
		return nil, fmt.Errorf("provider: schema, table, and index name required")
	}
	key := schemaName + "." + table + "." + indexName

	p.mu.Lock()
	defer p.mu.Unlock()

	if idx, ok := p.indexEngines[key]; ok {
		return idx, nil
	}

	idxTableName := fmt.Sprintf("%s__idx_%s", table, indexName)
	idx := btree.New(schemaName, idxTableName, p.tm, p.bm)
	p.indexEngines[key] = idx
	return idx, nil
}

// LoadCatalog initializes the schema catalog from system tables, creating them
// on first use if needed.
func (p *Provider) LoadCatalog(ctx context.Context) (*schema.Catalog, error) {
	tablesEng, err := p.Engine(schema.SystemSchema, schema.CatalogTablesTable)
	if err != nil {
		return nil, fmt.Errorf("provider: catalog tables engine: %w", err)
	}
	indexesEng, err := p.Engine(schema.SystemSchema, schema.CatalogIndexesTable)
	if err != nil {
		return nil, fmt.Errorf("provider: catalog indexes engine: %w", err)
	}

	cat := schema.NewCatalog(tablesEng, indexesEng)
	if err := cat.LoadSystemTables(ctx); err != nil {
		return nil, fmt.Errorf("provider: load catalog: %w", err)
	}
	return cat, nil
}

// Close releases file handles via the TableManager.
func (p *Provider) Close() error {
	if err := p.bm.FlushAll(); err != nil {
		return err
	}
	if err := p.wm.Close(); err != nil {
		return err
	}
	return p.tm.Close()
}

// FlushAll flushes dirty pages across all tables (implements txn.BufferFlusher).
func (p *Provider) FlushAll() error {
	return p.bm.FlushAll()
}

// SyncAll syncs all WAL loggers (implements txn.WALSyncer).
func (p *Provider) SyncAll() error {
	return p.wm.SyncAll()
}
