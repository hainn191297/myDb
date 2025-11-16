package heap

import (
	"errors"
	"fmt"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
)

// RecordID uniquely identifies a tuple (pageID + slot index).
type RecordID struct {
	Page page.PageID
	Slot uint16
}

type Table struct {
	Schema string
	Name   string

	bufMgr *engine.BufferManager
}

func NewTable(schema, name string, bm *engine.BufferManager) *Table {
	return &Table{
		Schema: schema,
		Name:   name,
		bufMgr: bm,
	}
}

func (t *Table) Insert(payload []byte) (RecordID, error) {
	if len(payload) == 0 {
		return RecordID{}, fmt.Errorf("heap: empty payload")
	}

	tm := t.bufMgr.TableManager()
	fm, err := tm.OpenTable(t.Schema, t.Name)
	if err != nil {
		return RecordID{}, err
	}

	totalPages, err := fm.NumPages()
	if err != nil {
		return RecordID{}, err
	}
	if totalPages == 0 {
		if _, err := t.allocateNewPage(fm); err != nil {
			return RecordID{}, err
		}
		totalPages = 1
	}

	for pid := page.PageID(1); pid <= page.PageID(totalPages); pid++ {
		pg, err := t.bufMgr.GetPage(t.Schema, t.Name, pid)
		if err != nil {
			return RecordID{}, err
		}

		if !pageInitialized(pg) {
			initPage(pg)
			t.bufMgr.MarkDirty(t.Schema, t.Name, pid)
		}

		slot, err := insertTuple(pg, payload)
		if err == nil {
			t.bufMgr.MarkDirty(t.Schema, t.Name, pid)
			t.bufMgr.Unpin(t.Schema, t.Name, pid)
			return RecordID{Page: pid, Slot: slot}, nil
		}
		t.bufMgr.Unpin(t.Schema, t.Name, pid)
		if !errors.Is(err, errNoSpace) {
			return RecordID{}, err
		}
	}

	newPID, err := t.allocateNewPage(fm)
	if err != nil {
		return RecordID{}, err
	}

	pg, err := t.bufMgr.GetPage(t.Schema, t.Name, newPID)
	if err != nil {
		return RecordID{}, err
	}
	if !pageInitialized(pg) {
		initPage(pg)
	}
	slot, err := insertTuple(pg, payload)
	if err != nil {
		t.bufMgr.Unpin(t.Schema, t.Name, newPID)
		return RecordID{}, err
	}
	t.bufMgr.MarkDirty(t.Schema, t.Name, newPID)
	t.bufMgr.Unpin(t.Schema, t.Name, newPID)
	return RecordID{Page: newPID, Slot: slot}, nil
}

func (t *Table) allocateNewPage(fm *page.FileManager) (page.PageID, error) {
	pid, err := fm.AllocatePage()
	if err != nil {
		return 0, err
	}
	p := page.NewPage(fmt.Sprintf("%s.%s", t.Schema, t.Name), pid)
	initPage(p)
	if err := fm.WritePage(p); err != nil {
		return 0, err
	}
	return pid, nil
}

// Delete marks the slot empty so space can be reused later.
func (t *Table) Delete(rid RecordID) error {
	pg, err := t.bufMgr.GetPage(t.Schema, t.Name, rid.Page)
	if err != nil {
		return err
	}
	defer t.bufMgr.Unpin(t.Schema, t.Name, rid.Page)

	slots, _, _ := readHeader(pg)
	if rid.Slot >= slots {
		return fmt.Errorf("heap: slot %d out of range", rid.Slot)
	}

	slot := getSlot(pg, rid.Slot)
	if slot.Length == 0 {
		return fmt.Errorf("heap: record already deleted")
	}

	clearSlot(pg, rid.Slot)
	t.bufMgr.MarkDirty(t.Schema, t.Name, rid.Page)
	return nil
}

// Scan iterates all tuples in page-slot order.
func (t *Table) Scan(fn func(RecordID, []byte) bool) error {
	tm := t.bufMgr.TableManager()
	fm, err := tm.OpenTable(t.Schema, t.Name)
	if err != nil {
		return err
	}

	totalPages, err := fm.NumPages()
	if err != nil {
		return err
	}

	for pid := page.PageID(1); pid <= page.PageID(totalPages); pid++ {
		pg, err := t.bufMgr.GetPage(t.Schema, t.Name, pid)
		if err != nil {
			return err
		}

		if !pageInitialized(pg) {
			t.bufMgr.Unpin(t.Schema, t.Name, pid)
			continue
		}

		stop := false
		iterateSlots(pg, func(slotID int, data []byte) bool {
			buf := make([]byte, len(data))
			copy(buf, data)
			cont := fn(RecordID{Page: pid, Slot: uint16(slotID)}, buf)
			if !cont {
				stop = true
			}
			return cont
		})
		t.bufMgr.Unpin(t.Schema, t.Name, pid)

		if stop {
			break
		}
	}

	return nil
}
