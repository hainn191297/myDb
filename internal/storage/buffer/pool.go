package buffer

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
BufferPool is the in-memory cache that holds a subset of disk pages
to reduce expensive disk I/O. It is one of the most important components
in a DBMS (similar to PostgreSQL, MySQL, SQLite).

Every entry in the buffer pool is a "frame". Each frame contains a Page
and metadata such as:

  - dirty bit:   whether the page was modified
  - pin count:   prevents eviction while in use
  - LRU position

When the buffer pool is full, the replacement strategy (LRU here)
selects a victim page to evict.
*/
type Pool struct {
	mu       sync.Mutex
	capacity int                           // max number of pages allowed in RAM
	frames   map[page.PageID]*list.Element // maps pageID to *list.Element
	lru      *list.List                    // LRU list of *Frame
	fm       *page.FileManager             // disk I/O manager
	freeList []*Frame                      // list of free frames
}

/*
Frame represents one slot in the buffer pool. It contains:

  - Page   → pointer to actual 4KB page
  - Dirty  → true if page was modified (must flush before eviction)
  - pinCnt → number of clients holding this page; cannot evict if > 0
*/
type Frame struct {
	Page   *page.Page
	Dirty  bool
	pinCnt int // number of clients using page, cannot evict if > 0
}

/*
NewPool creates a new BufferPool with the given capacity and FileManager.
*/
func NewPool(capacity int, fm *page.FileManager) *Pool {
	p := &Pool{
		capacity: capacity,
		frames:   make(map[page.PageID]*list.Element),
		lru:      list.New(),
		fm:       fm,
		freeList: make([]*Frame, 0, capacity),
	}

	// Pre-allocate empty frames (capacity number)
	for i := 0; i < capacity; i++ {
		p.freeList = append(p.freeList, &Frame{})
	}

	return p
}

/*
GetPage:
  - If page is cached → HIT → move to front → pin++
  - If page is not cached → MISS → read from disk → evict if needed → create frame
*/
func (p *Pool) Get(pageID page.PageID) (*page.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// cache hit
	if elem, ok := p.frames[pageID]; ok {

		frame := elem.Value.(*Frame)
		frame.pinCnt++ // same like cnt in shared pointer LOL; used to prevent eviction

		// move to front of LRU list
		p.lru.MoveToFront(elem)
		return frame.Page, nil
	}

	// cache miss: read page from disk
	page, err := p.fm.ReadPage(page.PageID(pageID))
	if err != nil {
		return nil, fmt.Errorf("buffer: read page %d: %w", pageID, err)
	}

	// if buffer pool is full, evict a page
	if p.lru.Len() >= p.capacity {
		if err := p.evict(); err != nil {
			return nil, err
		}
	}

	// insert new frame into buffer pool
	frame := &Frame{
		Page:   page,
		Dirty:  false,
		pinCnt: 1, // pin count starts at 1
	}
	elem := p.lru.PushFront(frame)
	p.frames[pageID] = elem

	return page, nil
}

/*
MarkDirty:

	Marks a page as modified so that eviction or Shutdown
	will flush the page back to disk.
*/
func (p *Pool) MarkDirty(pageID page.PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.frames[pageID]; ok {
		frame := elem.Value.(*Frame)
		frame.Dirty = true
	}
}

/*
Unpin:

	Decrements the pin count of a page, allowing it to be evicted
	if no clients are using it.
*/
func (p *Pool) Unpin(pageID page.PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.frames[pageID]; ok {
		frame := elem.Value.(*Frame)
		if frame.pinCnt > 0 {
			frame.pinCnt--
		}
	}
}

/*
FlushAll:

	Writes all dirty pages back to disk.
*/
func (p *Pool) FlushAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, elem := range p.frames {
		frame := elem.Value.(*Frame)
		if frame.Dirty {
			if err := p.fm.WritePage(frame.Page); err != nil {
				return fmt.Errorf("buffer: flush page %d: %w", id, err)
			}
			frame.Dirty = false
		}
	}

	return p.fm.Sync()
}

/*
evict:

	Selects a victim page using LRU and evicts it from the buffer pool.
	If the page is dirty, it is flushed to disk before eviction.
	Skips pages that are currently pinned (in use).
*/
func (p *Pool) evict() error {
	for e := p.lru.Back(); e != nil; e = e.Prev() {
		frame := e.Value.(*Frame)
		pageID := uint32(frame.Page.ID)

		// skip pinned pages
		if frame.pinCnt > 0 {
			continue
		}

		// flush if dirty
		if frame.Dirty {
			if err := p.fm.WritePage(frame.Page); err != nil {
				return fmt.Errorf("buffer: evict page %d: %w", pageID, err)
			}
		}

		// remove from buffer pool
		delete(p.frames, frame.Page.ID)
		p.lru.Remove(e)
		return nil
	}

	return fmt.Errorf("buffer: no unpinned pages to evict")
}
