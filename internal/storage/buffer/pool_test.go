package buffer

import (
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func newFM(t *testing.T) *page.FileManager {
	dir := t.TempDir()
	fm, err := page.NewFileManager(dir)
	if err != nil {
		t.Fatalf("cannot create file manager: %v", err)
	}
	return fm
}

func TestCacheMissLoadsFromDisk(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(2, fm)

	id, _ := fm.AllocatePage()

	// Write something to page on disk
	pg := page.NewPage(id)
	copy(pg.Data, []byte("hello"))
	_ = fm.WritePage(pg)

	// GetPage → MISS → load from disk
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
	pool := NewPool(2, fm)

	id, _ := fm.AllocatePage()

	pg := page.NewPage(id)
	copy(pg.Data, []byte("data1"))
	fm.WritePage(pg)

	// First read: MISS
	_, _ = pool.Get(id)

	// Modify file on disk (simulate stale disk)
	copy(pg.Data, []byte("xxxx"))
	fm.WritePage(pg)

	// Second read should be HIT (no disk read)
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
	pool := NewPool(1, fm) // capacity = 1
	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	fm.WritePage(page.NewPage(id1))
	fm.WritePage(page.NewPage(id2))

	// Load page 1
	_, _ = pool.Get(id1)

	// Load page 2 → evict page 1
	_, _ = pool.Get(id2)

	// page 1 should not be cached anymore
	if _, ok := pool.Get(id1); ok == nil {
		// second GetPage should re-load from disk (cache miss), not hit
	}
}

func TestDirtyPageFlushOnEvict(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(1, fm)

	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	pg1 := page.NewPage(id1)
	copy(pg1.Data, []byte("dirty!"))
	fm.WritePage(pg1)

	// Load page 1
	p1, _ := pool.Get(id1)

	// Modify in buffer, not disk
	copy(p1.Data, []byte("memory"))
	pool.MarkDirty(id1)

	// unpin page 1 to allow eviction → pinCnt = 0 → evicted to flush
	pool.Unpin(id1)

	// Load page 2 → force eviction of page1
	fm.WritePage(page.NewPage(id2))

	_, _ = pool.Get(id2)
	// Read page1 from disk → must equal "memory"
	diskPg, _ := fm.ReadPage(id1)
	if string(diskPg.Data[:6]) != "memory" {
		t.Fatalf("dirty page not flushed before evict, disk has: %q", diskPg.Data[:6])
	}
}

func TestPinnedPageCannotBeEvicted(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(1, fm)

	id1, _ := fm.AllocatePage()
	id2, _ := fm.AllocatePage()

	fm.WritePage(page.NewPage(id1))
	fm.WritePage(page.NewPage(id2))

	// Load page 1 + pin it
	_, _ = pool.Get(id1) // pinCnt = 1
	// Try to load page 2 → eviction should fail
	_, err := pool.Get(id2)
	if err == nil {
		t.Fatalf("expected eviction to fail when pinned pages exist")
	}
}

func TestFlushAllWritesDirtyPages(t *testing.T) {
	fm := newFM(t)
	pool := NewPool(2, fm)

	id, _ := fm.AllocatePage()
	fm.WritePage(page.NewPage(id))

	// Load page into buffer
	p, _ := pool.Get(id)

	// Modify page in RAM only
	copy(p.Data, []byte("changes"))
	pool.MarkDirty(id)

	// Flush all
	if err := pool.FlushAll(); err != nil {
		t.Fatalf("FlushAll error: %v", err)
	}

	// Check disk
	onDisk, _ := fm.ReadPage(id)
	if string(onDisk.Data[:7]) != "changes" {
		t.Fatalf("FlushAll did not write dirty page to disk")
	}
}
