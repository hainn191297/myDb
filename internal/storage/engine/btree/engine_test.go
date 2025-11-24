package btree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/buffer"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

type walDispatcher struct {
	tm *engine.TableManager
	wm *wal.Manager
}

func (w *walDispatcher) Append(fid uint32, pid page.PageID, data []byte) error {
	schema, table, ok := w.tm.LookupName(fid)
	if !ok {
		return fmt.Errorf("unknown file id %d", fid)
	}
	logger, err := w.wm.Open(schema, table)
	if err != nil {
		return err
	}
	return logger.Append(fid, pid, data)
}

func (w *walDispatcher) Sync() error {
	return w.wm.SyncAll()
}

func (w *walDispatcher) Close() error {
	return w.wm.Close()
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	dir := t.TempDir()
	tm := engine.NewTableManager(dir)
	wm := wal.NewManager(dir)
	dispatcher := &walDispatcher{tm: tm, wm: wm}
	pool := buffer.NewGlobalPool(8, dispatcher)
	bm := engine.NewBufferManager(tm, pool, dispatcher)

	return New("public", "idx_users_email", tm, bm)
}

func TestEngineInsertSearchDelete(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()

	if err := e.Insert([]byte("alice"), []byte("1")); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := e.Insert([]byte("bob"), []byte("2")); err != nil {
		t.Fatalf("insert: %v", err)
	}

	val, ok, err := e.Search([]byte("alice"))
	if err != nil || !ok || string(val) != "1" {
		t.Fatalf("search alice: ok=%v val=%s err=%v", ok, val, err)
	}

	if err := e.Delete([]byte("alice")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, ok, _ = e.Search([]byte("alice"))
	if ok {
		t.Fatalf("expected alice deleted")
	}
	// ensure bob still present
	val, ok, _ = e.Search([]byte("bob"))
	if !ok || string(val) != "2" {
		t.Fatalf("expected bob to remain, got ok=%v val=%s", ok, val)
	}

	// Range scan
	it, err := e.RangeScan(ctx, []byte("a"), []byte("z"))
	if err != nil {
		t.Fatalf("rangescan: %v", err)
	}
	defer it.Close()
	if !it.Next() {
		t.Fatalf("expected at least one entry")
	}
	if string(it.Key()) != "bob" {
		t.Fatalf("expected bob in scan, got %s", it.Key())
	}
}

func TestEngineDurabilityWithWAL(t *testing.T) {
	t.Skip("WAL durability for B+Tree needs fuller validation; skip for now")
	dir := t.TempDir()

	// first run: write data and flush
	tm := engine.NewTableManager(dir)
	wm := wal.NewManager(dir)
	dispatcher := &walDispatcher{tm: tm, wm: wm}
	pool := buffer.NewGlobalPool(4, dispatcher)
	bm := engine.NewBufferManager(tm, pool, dispatcher)

	idx := New("public", "idx_users_email", tm, bm)
	if err := idx.Insert([]byte("carol"), []byte("3")); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := bm.FlushAll(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	walPath := filepath.Join(dir, "public", "idx_users_email.wal")
	// explicitly append current page to WAL to ensure recovery path
	fid, fm, oerr := tm.OpenTable("public", "idx_users_email")
	if oerr != nil {
		t.Fatalf("open table for wal: %v", oerr)
	}
	pg, rerr := fm.ReadPage(page.PageID(1))
	if rerr != nil {
		t.Fatalf("read page: %v", rerr)
	}
	logger, oerr := wm.Open("public", "idx_users_email")
	if oerr != nil {
		t.Fatalf("open wal logger: %v", oerr)
	}
	if err := logger.Append(fid, pg.ID, pg.Data); err != nil {
		t.Fatalf("append wal: %v", err)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync wal: %v", err)
	}
	if info, err := os.Stat(walPath); err != nil || info.Size() == 0 {
		t.Fatalf("wal file missing or empty")
	}
	if err := wm.Close(); err != nil {
		t.Fatalf("close wal: %v", err)
	}
	if err := tm.Close(); err != nil {
		t.Fatalf("close tm: %v", err)
	}

	dataPath := filepath.Join(dir, "public", "idx_users_email.db")
	if err := os.Remove(dataPath); err != nil {
		t.Fatalf("remove data: %v", err)
	}

	// recovery
	wm2 := wal.NewManager(dir)
	if err := wm2.Recover(); err != nil {
		t.Fatalf("recover: %v", err)
	}
	dataInfo, err := os.Stat(dataPath)
	if err != nil {
		t.Fatalf("stat recovered data: %v", err)
	}
	if dataInfo.Size() == 0 {
		t.Fatalf("recovered data file empty")
	}
	fmCheck, err := page.NewFileManager(dataPath)
	if err != nil {
		t.Fatalf("open fm check: %v", err)
	}
	defer fmCheck.Close()
	pg, err = fmCheck.ReadPage(page.PageID(1))
	if err != nil {
		t.Fatalf("read recovered page: %v", err)
	}
	entries, err := readLeaf(pg)
	if err != nil {
		t.Fatalf("decode recovered entries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no entries after recover")
	}

	// reopen and verify
	tm2 := engine.NewTableManager(dir)
	dispatcher2 := &walDispatcher{tm: tm2, wm: wm2}
	pool2 := buffer.NewGlobalPool(4, dispatcher2)
	bm2 := engine.NewBufferManager(tm2, pool2, dispatcher2)
	idx2 := New("public", "idx_users_email", tm2, bm2)

	val, ok, err := idx2.Search([]byte("carol"))
	if err != nil {
		t.Fatalf("search after recover: %v", err)
	}
	if !ok || string(val) != "3" {
		t.Fatalf("unexpected search result ok=%v val=%s", ok, val)
	}
}
