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

	writePage func(*page.Page) error
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
	writer func(*page.Page) error,
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
	f.writePage = writer

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
	if dirty {
		f.Dirty = true
	}
}

func (g *GlobalPool) evict() (*Frame, error) {
	for e := g.lru.Back(); e != nil; e = e.Prev() {
		f := e.Value.(*Frame)

		if f.pinCnt > 0 {
			continue
		}

		if f.Dirty {
			if err := g.flushFrame(f); err != nil {
				return nil, err
			}
		}

		delete(g.frames, PageKey{FileID: f.FileID, PID: f.PID})
		g.lru.Remove(e)

		// reset frame
		f.Page = nil
		f.Dirty = false
		f.pinCnt = 0
		f.FileID = 0
		f.PID = 0
		f.writePage = nil

		return f, nil
	}
	return nil, fmt.Errorf("globalpool: no clean unpinned frame to evict")
}

func (g *GlobalPool) flushFrame(f *Frame) error {
	if f.Page == nil {
		return fmt.Errorf("globalpool: no page data to flush")
	}
	if f.writePage == nil {
		return fmt.Errorf("globalpool: missing writer for file %d page %d", f.FileID, f.PID)
	}
	if g.wal != nil {
		if err := g.wal.Append(f.FileID, f.PID, f.Page.Data); err != nil {
			return fmt.Errorf("globalpool: wal append on evict: %w", err)
		}
		if err := g.wal.Sync(); err != nil {
			return fmt.Errorf("globalpool: wal sync on evict: %w", err)
		}
	}
	if err := f.writePage(f.Page); err != nil {
		return fmt.Errorf("globalpool: write page fid=%d pid=%d: %w", f.FileID, f.PID, err)
	}
	f.Dirty = false
	return nil
}

/*
FlushMatching flushes every dirty frame whose FileID satisfies shouldFlush.
If shouldFlush == nil → flush all dirty pages.
*/
func (g *GlobalPool) FlushMatching(
	shouldFlush func(fileID uint32) bool,
	writePage func(fileID uint32, p *page.Page) error,
	syncFile func(fileID uint32) error,
) error {

	g.mu.Lock()
	defer g.mu.Unlock()

	// gom frame dirty
	var dirtyFrames []*Frame
	seenFiles := make(map[uint32]bool)
	matches := func(fid uint32) bool {
		if shouldFlush == nil {
			return true
		}
		return shouldFlush(fid)
	}

	// WAL append cho tất cả dirty pages
	if g.wal != nil {
		for _, elem := range g.frames {
			f := elem.Value.(*Frame)
			if f.Dirty && f.Page != nil && matches(f.FileID) {
				dirtyFrames = append(dirtyFrames, f)
				if err := g.wal.Append(f.FileID, f.PID, f.Page.Data); err != nil {
					return fmt.Errorf("globalpool: wal append on flush: %w", err)
				}
			}
		}
		if len(dirtyFrames) > 0 {
			if err := g.wal.Sync(); err != nil {
				return fmt.Errorf("globalpool: wal sync on flush: %w", err)
			}
		}
	} else {
		// không dùng WAL → vẫn cần biết frame nào dirty để ghi
		for _, elem := range g.frames {
			f := elem.Value.(*Frame)
			if f.Dirty && f.Page != nil && matches(f.FileID) {
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

/*
FlushAll flushes every dirty page regardless of fileID.
*/
func (g *GlobalPool) FlushAll(
	writePage func(fileID uint32, p *page.Page) error,
	syncFile func(fileID uint32) error,
) error {
	return g.FlushMatching(nil, writePage, syncFile)
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
