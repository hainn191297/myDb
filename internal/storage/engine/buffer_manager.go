package engine

import (
	"fmt"

	"github.com/hainn191297/myDb/internal/storage/buffer"
	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

/*
BufferManager is the layer that:

  - Resolves schema/table → FileID + FileManager (via TableManager)
  - Loads pages via GlobalPool (which handles LRU + WAL)
  - Exposes simple page-level API to storage engines (heap, btree)

This makes the upper engines completely independent of I/O layout.
*/
type BufferManager struct {
	tm   *TableManager
	pool *buffer.GlobalPool
	wal  wal.Logger
}

func NewBufferManager(tm *TableManager, pool *buffer.GlobalPool, walLogger wal.Logger) *BufferManager {
	return &BufferManager{
		tm:   tm,
		pool: pool,
		wal:  walLogger,
	}
}

/*
GetPage returns (fileID, *Page):

  - TableManager.OpenTable → (fid, fm)
  - GlobalPool.GetPage(fid, pid, reader)
*/
func (bm *BufferManager) GetPage(
	schema, table string,
	pid page.PageID,
) (uint32, *page.Page, error) {

	fid, fm, err := bm.tm.OpenTable(schema, table)
	if err != nil {
		return 0, nil, err
	}

	reader := func(pid page.PageID) (*page.Page, error) {
		return fm.ReadPage(pid)
	}

	pg, err := bm.pool.GetPage(fid, pid, reader)
	if err != nil {
		return 0, nil, err
	}

	return fid, pg, nil
}

/*
Unpin:

  - dirty=false → unpin only
  - dirty=true  → unpin + mark dirty
*/
func (bm *BufferManager) Unpin(fid uint32, pid page.PageID, dirty bool) {
	bm.pool.Unpin(fid, pid, dirty)
}

/*
MarkDirty is a convenience helper.
(NOTE: MarkDirty does NOT unpin!)
*/
func (bm *BufferManager) MarkDirty(fid uint32, pid page.PageID) {
	bm.pool.MarkDirty(fid, pid)
}

/*
FlushTable flushes all dirty pages for this specific table.
*/
func (bm *BufferManager) FlushTable(schema, table string) error {
	fid, fm, err := bm.tm.OpenTable(schema, table)
	if err != nil {
		return err
	}

	writeFn := func(fileID uint32, p *page.Page) error {
		if fileID != fid {
			return nil // skip pages from other files
		}
		return fm.WritePage(p)
	}

	syncFn := func(fileID uint32) error {
		if fileID == fid {
			return fm.Sync()
		}
		return nil
	}

	return bm.pool.FlushAll(writeFn, syncFn)
}

/*
FlushAll flushes all dirty pages across ALL FileManagers.
*/
func (bm *BufferManager) FlushAll() error {
	writeFn := func(fid uint32, p *page.Page) error {
		fm, ok := bm.tm.LookupByFileID(fid)
		if !ok {
			return fmt.Errorf("buffer: missing FileManager for fid=%d", fid)
		}
		return fm.WritePage(p)
	}

	syncFn := func(fid uint32) error {
		fm, ok := bm.tm.LookupByFileID(fid)
		if !ok {
			return fmt.Errorf("buffer: missing FileManager for fid=%d", fid)
		}
		return fm.Sync()
	}

	return bm.pool.FlushAll(writeFn, syncFn)
}
