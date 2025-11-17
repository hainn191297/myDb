package heap

import (
	"fmt"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
Table represents one heap-organized table file.

It does NOT know schema of tuples; it only stores opaque []byte blobs
(encoding/decoding is handled by tuple.go / heap engine).

Responsibilities:
  - choose a page with enough free space for inserts
  - allocate new pages when needed
  - iterate over all tuples (full scan)
  - delete tuple by (pageID, slotIdx)
*/
type Table struct {
	schema string
	name   string

	tm *engine.TableManager
	bm *engine.BufferManager
}

// NewTable creates a heap table handle bound to a schema/table name.
func NewTable(schema, name string, tm *engine.TableManager, bm *engine.BufferManager) *Table {
	return &Table{
		schema: schema,
		name:   name,
		tm:     tm,
		bm:     bm,
	}
}

/*
Insert appends a new tuple into this heap table.

It scans existing pages for enough free space; if none found,
it allocates a new physical page via FileManager, initializes it
as an empty heap page, and inserts there.

RETURN:
  - pageID where tuple is stored
  - slot index inside that page
*/
func (t *Table) Insert(tuple []byte) (page.PageID, int, error) {
	// Open underlying file to know how many pages exist
	fid, fm, err := t.tm.OpenTable(t.schema, t.name)
	if err != nil {
		return 0, -1, err
	}

	numPages, err := fm.NumPages()
	if err != nil {
		return 0, -1, fmt.Errorf("heap: NumPages failed: %w", err)
	}

	// Try to find an existing page with enough free space
	for pid := page.PageID(1); pid <= page.PageID(numPages); pid++ {
		_, raw, err := t.bm.GetPage(t.schema, t.name, pid)
		if err != nil {
			return 0, -1, fmt.Errorf("heap: GetPage(%d): %w", pid, err)
		}

		hp := WrapPage(raw)
		need := len(tuple) + 2 // tuple bytes + 1 slot entry (2 bytes)

		if hp.FreeSpace() >= need {
			slot, ierr := hp.InsertTuple(tuple)
			// mark page dirty & unpin
			t.bm.Unpin(fid, pid, ierr == nil)
			if ierr != nil {
				return 0, -1, ierr
			}
			return pid, slot, nil
		}

		// not enough space: unpin clean
		t.bm.Unpin(fid, pid, false)
	}

	// No existing page has room → allocate a new page
	newPID, err := fm.AllocatePage()
	if err != nil {
		return 0, -1, fmt.Errorf("heap: allocate page failed: %w", err)
	}

	// initialize as empty heap page and persist initial header
	hp := NewEmptyPage(newPID)
	if err := fm.WritePage(hp.Raw()); err != nil {
		return 0, -1, fmt.Errorf("heap: write new page %d: %w", newPID, err)
	}

	// load via buffer manager so buffer pool knows about it
	_, raw, err := t.bm.GetPage(t.schema, t.name, newPID)
	if err != nil {
		return 0, -1, fmt.Errorf("heap: GetPage(new %d): %w", newPID, err)
	}
	hp2 := WrapPage(raw)

	slot, ierr := hp2.InsertTuple(tuple)
	t.bm.Unpin(fid, newPID, ierr == nil)
	if ierr != nil {
		return 0, -1, ierr
	}

	return newPID, slot, nil
}

/*
Scan calls fn(pid, slotIdx, offset) for every LIVE tuple in the table.

- pid      = physical page ID
- slotIdx  = slot index within that page
- offset   = byte offset inside the page where tuple bytes begin

fn can return false to stop early (for e.g. point lookups).

Decoding of tuple bytes is handled at a higher layer (tuple.go).
*/
func (t *Table) Scan(fn func(pid page.PageID, slot int, offset uint16, pg *page.Page) bool) error {
	// Need NumPages() → ask underlying FileManager
	fid, fm, err := t.tm.OpenTable(t.schema, t.name)
	if err != nil {
		return err
	}

	numPages, err := fm.NumPages()
	if err != nil {
		return fmt.Errorf("heap: NumPages failed: %w", err)
	}

	for pid := page.PageID(1); pid <= page.PageID(numPages); pid++ {
		_, raw, err := t.bm.GetPage(t.schema, t.name, pid)
		if err != nil {
			return fmt.Errorf("heap: GetPage(%d): %w", pid, err)
		}

		hp := WrapPage(raw)
		stop := false

		hp.ForEach(func(slot int, off uint16) bool {
			if !fn(pid, slot, off, raw) {
				stop = true
				return false
			}
			return true
		})

		// this page may be read-only during scan
		t.bm.Unpin(fid, pid, false)

		if stop {
			break
		}
	}

	return nil
}

/*
Delete deletes a tuple by (pageID, slotIdx).

This only marks the slot as free on that page (no compaction yet).
*/
func (t *Table) Delete(pid page.PageID, slot int) error {
	fid, _, err := t.tm.OpenTable(t.schema, t.name)
	if err != nil {
		return err
	}

	_, raw, err := t.bm.GetPage(t.schema, t.name, pid)
	if err != nil {
		return fmt.Errorf("heap: GetPage(%d): %w", pid, err)
	}

	hp := WrapPage(raw)
	if err := hp.DeleteTuple(slot); err != nil {
		t.bm.Unpin(fid, pid, false)
		return err
	}

	t.bm.Unpin(fid, pid, true)
	return nil
}
