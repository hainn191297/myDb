package buffer

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

// PageKey uniquely identifies a page in the global cache.
type PageKey struct {
	FileID uint32
	PID    page.PageID
}

// Frame = one slot in the global buffer pool.
type Frame struct {
	Page   *page.Page
	Dirty  bool
	pinCnt int

	FileID uint32
	PID    page.PageID
}

// GlobalPool is a single LRU-based buffer pool shared by all files.
type GlobalPool struct {
	mu       sync.Mutex
	capacity int

	frames map[PageKey]*list.Element // key -> *Frame (wrapped in list.Element)
	lru    *list.List                // front = MRU, back = LRU
	free   []*Frame                  // preallocated empty frames

	wal wal.Logger // may be nil for testing
}

// NewGlobalPool preallocates all frame slots.
func NewGlobalPool(capacity int, walLogger wal.Logger) *GlobalPool {
	g := &GlobalPool{
		capacity: capacity,
		frames:   make(map[PageKey]*list.Element),
		lru:      list.New(),
		free:     make([]*Frame, 0, capacity),
		wal:      walLogger,
	}
	for i := 0; i < capacity; i++ {
		g.free = append(g.free, &Frame{})
	}
	return g
}

/*
GetPage loads or returns a cached page:

 1. Cache hit  → pin++, move frame to LRU front, return existing *page.Page
 2. Cache miss → choose frame (free or evict LRU unpinned), read from disk via reader,
    insert into LRU front, pin=1, return *page.Page

reader(pid) phải đọc page từ FileManager tương ứng (do BufferManager truyền vào).
*/
func (g *GlobalPool) GetPage(
	fileID uint32,
	pid page.PageID,
	reader func(page.PageID) (*page.Page, error),
) (*page.Page, error) {

	g.mu.Lock()
	defer g.mu.Unlock()

	key := PageKey{FileID: fileID, PID: pid}

	// CACHE HIT
	if elem, ok := g.frames[key]; ok {
		f := elem.Value.(*Frame)
		f.pinCnt++
		g.lru.MoveToFront(elem)
		return f.Page, nil
	}

	// CACHE MISS → read from disk
	pg, err := reader(pid)
	if err != nil {
		return nil, fmt.Errorf("globalpool: read page %v: %w", pid, err)
	}

	// Choose a frame: free list first, otherwise evict LRU unpinned
	var f *Frame
	if n := len(g.free); n > 0 {
		f = g.free[n-1]
		g.free = g.free[:n-1]
	} else {
		f, err = g.evict()
		if err != nil {
			return nil, err
		}
	}

	// Assign new mapping
	f.Page = pg
	f.Dirty = false
	f.pinCnt = 1
	f.FileID = fileID
	f.PID = pid

	elem := g.lru.PushFront(f)
	g.frames[key] = elem

	return pg, nil
}

func (g *GlobalPool) Unpin(fileID uint32, pid page.PageID, dirty bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := PageKey{FileID: fileID, PID: pid}
	elem, ok := g.frames[key]
	if !ok {
		return
	}

	f := elem.Value.(*Frame)
	if f.pinCnt > 0 {
		f.pinCnt--
	}
}

func (g *GlobalPool) evict() (*Frame, error) {
	for e := g.lru.Back(); e != nil; e = e.Prev() {
		f := e.Value.(*Frame)

		// Chỉ evict frame hoàn toàn clean + unpinned
		if f.pinCnt > 0 || f.Dirty {
			continue
		}

		delete(g.frames, PageKey{FileID: f.FileID, PID: f.PID})
		g.lru.Remove(e)

		// reset frame
		f.Page = nil
		f.Dirty = false
		f.pinCnt = 0

		return f, nil
	}
	return nil, fmt.Errorf("globalpool: no clean unpinned frame to evict")
}

/*
FlushAll:

  - Duyệt toàn bộ frames
  - Với frame Dirty:
  - WAL.Append(fileID, pid, data)
  - WAL.Sync()
  - Gọi writePage(fid, page) cho từng frame dirty
  - Gọi syncFile(fid) 1 lần cho mỗi fileID bị ảnh hưởng

writePage/syncFile do BufferManager/engine truyền vào
để ánh xạ ngược FileID → FileManager tương ứng.
*/
func (g *GlobalPool) FlushAll(
	writePage func(fileID uint32, p *page.Page) error,
	syncFile func(fileID uint32) error,
) error {

	g.mu.Lock()
	defer g.mu.Unlock()

	// gom frame dirty
	var dirtyFrames []*Frame
	seenFiles := make(map[uint32]bool)

	// WAL append cho tất cả dirty pages
	if g.wal != nil {
		for _, elem := range g.frames {
			f := elem.Value.(*Frame)
			if f.Dirty && f.Page != nil {
				dirtyFrames = append(dirtyFrames, f)
				if err := g.wal.Append(f.FileID, f.PID, f.Page.Data); err != nil {
					return fmt.Errorf("globalpool: wal append on flush: %w", err)
				}
			}
		}
		if err := g.wal.Sync(); err != nil {
			return fmt.Errorf("globalpool: wal sync on flush: %w", err)
		}
	} else {
		// không dùng WAL → vẫn cần biết frame nào dirty để ghi
		for _, elem := range g.frames {
			f := elem.Value.(*Frame)
			if f.Dirty && f.Page != nil {
				dirtyFrames = append(dirtyFrames, f)
			}
		}
	}

	// Ghi page về disk
	for _, f := range dirtyFrames {
		if err := writePage(f.FileID, f.Page); err != nil {
			return fmt.Errorf("globalpool: writePage(fid=%d, pid=%d): %w", f.FileID, f.PID, err)
		}
		f.Dirty = false
		seenFiles[f.FileID] = true
	}

	// Sync từng file 1 lần
	for fid := range seenFiles {
		if err := syncFile(fid); err != nil {
			return fmt.Errorf("globalpool: syncFile(fid=%d): %w", fid, err)
		}
	}

	return nil
}

func (g *GlobalPool) MarkDirty(fileID uint32, pid page.PageID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := PageKey{FileID: fileID, PID: pid}
	elem, ok := g.frames[key]
	if !ok {
		return
	}

	f := elem.Value.(*Frame)
	f.Dirty = true
}
