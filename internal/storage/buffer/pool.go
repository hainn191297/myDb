package buffer

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

/*
BufferPool manages a fixed-size in-memory cache of pages.
Each Pool operates on the FileManager supplied at construction time
(TableManager typically hands out one FileManager per table file).

Main components:
- frames:   map[PageID]*list.Element     → active frames
- lru:      doubly-linked list of *Frame  → eviction order
- freeList: pool of empty frames
- fm:       FileManager → performs disk I/O for that table file
*/
type Pool struct {
	mu       sync.Mutex
	capacity int

	frames map[page.PageID]*list.Element
	lru    *list.List // front = most recently used, back = LRU

	freeList []*Frame // preallocated empty frames
	fm       *page.FileManager

	table string
	wal   wal.Logger
}

/*
Frame = one buffer pool slot.

Reset rules:
- When evicted, frame goes back to freeList.
- LRU only contains frames that are actively mapped to a page.
*/
type Frame struct {
	Page   *page.Page
	Dirty  bool
	pinCnt int
}

/*
NewPool preallocates all frame slots and stores them inside freeList.
This avoids GC overhead and mimics InnoDB/Postgres architecture.
*/
func NewPool(capacity int, fm *page.FileManager, table string, logger wal.Logger) *Pool {
	p := &Pool{
		capacity: capacity,
		frames:   make(map[page.PageID]*list.Element),
		lru:      list.New(),
		freeList: make([]*Frame, 0, capacity),
		fm:       fm,
		table:    table,
		wal:      logger,
	}

	// Preallocate frames as empty slots
	for i := 0; i < capacity; i++ {
		p.freeList = append(p.freeList, &Frame{})
	}

	return p
}

/*
Get(id) loads a page into buffer pool:

1. Cache hit  → return frame, pin++, move to LRU front
2. Cache miss → load from disk, use free frame OR evict, insert to LRU front
*/
func (p *Pool) Get(pid page.PageID) (*page.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// HIT
	if elem, ok := p.frames[pid]; ok {
		f := elem.Value.(*Frame)
		f.pinCnt++
		p.lru.MoveToFront(elem)
		return f.Page, nil
	}

	// MISS → load from disk
	pg, err := p.fm.ReadPage(pid)
	if err != nil {
		return nil, fmt.Errorf("buffer: read page %v: %w", pid, err)
	}
	pg.Table = p.table

	// choose frame
	var frame *Frame
	if len(p.freeList) > 0 {
		idx := len(p.freeList) - 1
		frame = p.freeList[idx]
		p.freeList = p.freeList[:idx]
	} else {
		frame, err = p.evict()
		if err != nil {
			return nil, err
		}
	}

	frame.Page = pg
	frame.Dirty = false
	frame.pinCnt = 1

	elem := p.lru.PushFront(frame)
	p.frames[pid] = elem

	return pg, nil
}

func (p *Pool) MarkDirty(pid page.PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.frames[pid]; ok {
		elem.Value.(*Frame).Dirty = true
	}
}

func (p *Pool) Unpin(pid page.PageID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.frames[pid]; ok {
		f := elem.Value.(*Frame)
		if f.pinCnt > 0 {
			f.pinCnt--
		}
	}
}

func (p *Pool) FlushAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for pid, elem := range p.frames {
		f := elem.Value.(*Frame)
		if f.Dirty {
			if err := p.writePage(f.Page); err != nil {
				return fmt.Errorf("flush page %v: %w", pid, err)
			}
			f.Dirty = false
		}
	}

	if p.wal != nil {
		if err := p.wal.Sync(); err != nil {
			return err
		}
	}

	return p.fm.Sync()
}

/*
evict() selects the least-recently-used *unpinned* frame,
flushes if dirty, removes it from structures, resets it,
and then returns it so Get() can reassign it.

Flow:
  - iterate from LRU tail
  - skip pinned frames
  - flush dirty pages
  - remove map + LRU
  - reset frame, push to freeList
*/
func (p *Pool) evict() (*Frame, error) {
	for e := p.lru.Back(); e != nil; e = e.Prev() {
		f := e.Value.(*Frame)

		if f.pinCnt > 0 {
			continue
		}

		if f.Dirty {
			if err := p.writePage(f.Page); err != nil {
				return nil, fmt.Errorf("evict flush page %v: %w", f.Page.ID, err)
			}
		}

		delete(p.frames, f.Page.ID)
		p.lru.Remove(e)

		// reset frame
		f.Page = nil
		f.Dirty = false
		f.pinCnt = 0

		return f, nil
	}

	return nil, fmt.Errorf("buffer: no unpinned pages to evict")
}

func (p *Pool) writePage(pg *page.Page) error {
	if p.wal != nil {
		if err := p.wal.Append(p.table, pg.ID, pg.Data); err != nil {
			return err
		}
	}
	return p.fm.WritePage(pg)
}
