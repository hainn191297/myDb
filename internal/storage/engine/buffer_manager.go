package engine

import (
	"fmt"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/buffer"
	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

/*
BufferManager coordinates TableManager + BufferPool.
It lazily creates a buffer.Pool per (schema, table) combination so higher
layers can fetch pages without opening FileManagers manually.
*/
type BufferManager struct {
	tableMgr *TableManager
	walMgr   *wal.Manager
	capacity int

	mu    sync.Mutex
	pools map[string]*buffer.Pool
}

func NewBufferManager(tm *TableManager, walMgr *wal.Manager, capacity int) *BufferManager {
	return &BufferManager{
		tableMgr: tm,
		walMgr:   walMgr,
		capacity: capacity,
		pools:    make(map[string]*buffer.Pool),
	}
}

func (bm *BufferManager) poolKey(schema, table string) string {
	return schema + "/" + table
}

// TableManager exposes the underlying table manager (needed for heap).
func (bm *BufferManager) TableManager() *TableManager {
	return bm.tableMgr
}

/*
getPool returns (or constructs) the buffer pool for schema.table.
*/
func (bm *BufferManager) getPool(schema, table string) (*buffer.Pool, error) {
	key := bm.poolKey(schema, table)

	bm.mu.Lock()
	defer bm.mu.Unlock()

	if pool, ok := bm.pools[key]; ok {
		return pool, nil
	}

	fm, err := bm.tableMgr.OpenTable(schema, table)
	if err != nil {
		return nil, fmt.Errorf("open table %s.%s: %w", schema, table, err)
	}

	var logger wal.Logger
	if bm.walMgr != nil {
		logger, err = bm.walMgr.Open(schema, table)
		if err != nil {
			return nil, fmt.Errorf("open wal %s.%s: %w", schema, table, err)
		}
	}

	pool := buffer.NewPool(bm.capacity, fm, bm.poolKey(schema, table), logger)
	bm.pools[key] = pool
	return pool, nil
}

func (bm *BufferManager) GetPage(schema, table string, pid page.PageID) (*page.Page, error) {
	pool, err := bm.getPool(schema, table)
	if err != nil {
		return nil, err
	}
	return pool.Get(pid)
}

/*
MarkDirty flips the dirty flag for the given page inside its table pool.
*/
func (bm *BufferManager) MarkDirty(schema, table string, pid page.PageID) error {
	pool, err := bm.getPool(schema, table)
	if err != nil {
		return err
	}
	pool.MarkDirty(pid)
	return nil
}

/*
Unpin decrements the pin count so the page can become evictable.
*/
func (bm *BufferManager) Unpin(schema, table string, pid page.PageID) error {
	pool, err := bm.getPool(schema, table)
	if err != nil {
		return err
	}
	pool.Unpin(pid)
	return nil
}

/*
FlushTable writes every dirty page belonging to schema.table.
*/
func (bm *BufferManager) FlushTable(schema, table string) error {
	pool, err := bm.getPool(schema, table)
	if err != nil {
		return err
	}
	return pool.FlushAll()
}
