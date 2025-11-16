package buffer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

func newFM(t *testing.T) *page.FileManager {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "table.db")
	fm, err := page.NewFileManager(file)
	if err != nil {
		t.Fatalf("cannot create file manager: %v", err)
	}
	return fm
}

func TestCacheMissLoadsFromDisk(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(2, fm, "table.db", nil)

	id, _ := fm.AllocatePage()

	pg := page.NewPage("table.db", id)
	copy(pg.Data, []byte("hello"))
	fm.WritePage(pg)

	// MISS → load from disk
	got, err := pool.Get(id)
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}

	if string(got.Data[:5]) != "hello" {
		t.Fatalf("expected 'hello', got %q", got.Data[:5])
	}
}

func TestCacheHitDoesNotReadFromDiskAgain(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(2, fm, "table.db", nil)

	id, _ := fm.AllocatePage()

	pg := page.NewPage("table.db", id)
	copy(pg.Data, []byte("data1"))
	fm.WritePage(pg)

	// First read: load into RAM
	_, _ = pool.Get(id)

	// Modify disk to simulate stale disk content
	copy(pg.Data, []byte("xxxx"))
	fm.WritePage(pg)

	// Second read MUST be HIT → must still get "data1"
	got, err := pool.Get(id)
	if err != nil {
		t.Fatalf("GetPage error: %v", err)
	}

	if string(got.Data[:5]) != "data1" {
		t.Fatalf("cache hit should not reload from disk, got: %q", got.Data[:5])
	}
}

func TestEvictionOccursWhenPoolFull(t *testing.T) {
	fm := newFM(t)

	// capacity=1 but freeList also has 1 preallocated frame
	pool := NewPool(1, fm, "table.db", nil)

	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	fm.WritePage(page.NewPage("table.db", id1))
	fm.WritePage(page.NewPage("table.db", id2))

	// Load page1 → occupy a frame
	_, _ = pool.Get(id1)

	// Now freeList is empty, next load MUST evict
	pool.Unpin(id1) // allow eviction

	_, err := pool.Get(id2)
	if err != nil {
		t.Fatalf("unexpected eviction error: %v", err)
	}

	// page1 should have been removed from frames map
	if _, ok := pool.frames[id1]; ok {
		t.Fatalf("page1 should have been evicted from buffer")
	}

	// Unpin page2 so pool can reuse the slot
	pool.Unpin(id2)

	// Reload page1 to ensure miss path still works
	if _, err := pool.Get(id1); err != nil {
		t.Fatalf("expected to reload evicted page: %v", err)
	}
}

func TestDirtyPageFlushOnEvict(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(1, fm, "table.db", nil)

	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	pg1 := page.NewPage("table.db", id1)
	copy(pg1.Data, []byte("dirty!"))
	fm.WritePage(pg1)

	// Load page 1
	p1, _ := pool.Get(id1)

	// Modify in RAM only
	copy(p1.Data, []byte("memory"))
	pool.MarkDirty(id1)

	// allow eviction
	pool.Unpin(id1)

	// Load page 2 → evict page1
	fm.WritePage(page.NewPage("table.db", id2))
	_, _ = pool.Get(id2)

	// Must have flushed "memory"
	diskPg, _ := fm.ReadPage(id1)
	if string(diskPg.Data[:6]) != "memory" {
		t.Fatalf("dirty page not flushed, disk has: %q", diskPg.Data[:6])
	}
}

func TestPinnedPageCannotBeEvicted(t *testing.T) {
	fm := newFM(t)

	// capacity=1 → freeList has 1 frame only
	pool := NewPool(1, fm, "table.db", nil)

	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	fm.WritePage(page.NewPage("table.db", id1))
	fm.WritePage(page.NewPage("table.db", id2))

	// Load page1 (pinCnt=1)
	_, _ = pool.Get(id1)

	// freeList empty now; next Get MUST try eviction
	_, err := pool.Get(id2)

	if err == nil {
		t.Fatalf("expected eviction to fail when pinned frames exist")
	}
}

func TestFlushAllWritesDirtyPages(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(2, fm, "table.db", nil)

	id, _ := fm.AllocatePage()
	fm.WritePage(page.NewPage("table.db", id))

	// Load
	p, _ := pool.Get(id)

	// Modify RAM only
	copy(p.Data, []byte("changes"))
	pool.MarkDirty(id)

	// Flush all dirty pages
	if err := pool.FlushAll(); err != nil {
		t.Fatalf("FlushAll error: %v", err)
	}

	onDisk, _ := fm.ReadPage(id)
	if string(onDisk.Data[:7]) != "changes" {
		t.Fatalf("FlushAll did not write dirty page to disk")
	}
}

func TestFlushAllAppendsToWAL(t *testing.T) {
	fm := newFM(t)

	dir := t.TempDir()
	logPath := filepath.Join(dir, "table.wal")
	logger, err := wal.OpenLogger(logPath)
	if err != nil {
		t.Fatalf("open wal: %v", err)
	}
	defer logger.Close()

	pool := NewPool(1, fm, "public.table", logger)

	id, _ := fm.AllocatePage()
	fm.WritePage(page.NewPage("public.table", id))

	p, _ := pool.Get(id)
	copy(p.Data, []byte("abcd"))
	pool.MarkDirty(id)

	if err := pool.FlushAll(); err != nil {
		t.Fatalf("FlushAll: %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat wal: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("wal file should have records")
	}
}
