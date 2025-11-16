package heap

import (
	"bytes"
	"context"
	"fmt"

	enginepkg "github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
)

// Engine exposes a heap-backed implementation of engine.Engine.
type Engine struct {
	table *Table
}

func NewEngine(schema, name string, bm *enginepkg.BufferManager) *Engine {
	return &Engine{table: NewTable(schema, name, bm)}
}

func (e *Engine) Put(ctx context.Context, key, value []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("heap engine: empty key")
	}

	if err := e.removeIfExists(key); err != nil {
		return err
	}
	_, err := e.table.Insert(encodeTuple(key, value))
	return err
}

func (e *Engine) Get(ctx context.Context, key []byte) ([]byte, error) {
	var (
		found   []byte
		scanErr error
	)
	err := e.table.Scan(func(_ RecordID, data []byte) bool {
		k, v, err := decodeTuple(data)
		if err != nil {
			scanErr = err
			return false
		}
		if bytes.Equal(k, key) {
			found = append([]byte(nil), v...)
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if found == nil {
		return nil, fmt.Errorf("heap engine: key not found")
	}
	return found, nil
}

func (e *Engine) Delete(ctx context.Context, key []byte) error {
	rid, err := e.findRecord(key)
	if err != nil {
		return err
	}
	if rid == nil {
		return fmt.Errorf("heap engine: key not found")
	}
	return e.table.Delete(*rid)
}

func (e *Engine) Scan(ctx context.Context, start, end []byte) (enginepkg.Iterator, error) {
	it, err := newIterator(e.table, start, end)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (e *Engine) removeIfExists(key []byte) error {
	rid, err := e.findRecord(key)
	if err != nil {
		return err
	}
	if rid == nil {
		return nil
	}
	return e.table.Delete(*rid)
}

func (e *Engine) findRecord(key []byte) (*RecordID, error) {
	var (
		match   *RecordID
		scanErr error
	)
	err := e.table.Scan(func(rid RecordID, data []byte) bool {
		k, _, err := decodeTuple(data)
		if err != nil {
			scanErr = err
			return false
		}
		if bytes.Equal(k, key) {
			tmp := rid
			match = &tmp
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if scanErr != nil {
		return nil, scanErr
	}
	return match, nil
}

type iterator struct {
	table    *Table
	start    []byte
	end      []byte
	total    page.PageID
	nextPID  page.PageID
	curPID   page.PageID
	page     *page.Page
	slots    uint16
	slotIdx  uint16
	key      []byte
	value    []byte
	err      error
	finished bool
}

func newIterator(table *Table, start, end []byte) (*iterator, error) {
	fm, err := table.bufMgr.TableManager().OpenTable(table.Schema, table.Name)
	if err != nil {
		return nil, err
	}
	totalPages, err := fm.NumPages()
	if err != nil {
		return nil, err
	}
	return &iterator{
		table:   table,
		start:   append([]byte(nil), start...),
		end:     append([]byte(nil), end...),
		total:   page.PageID(totalPages),
		nextPID: 1,
	}, nil
}

func (it *iterator) Next() bool {
	if it.err != nil || it.finished {
		return false
	}
	for {
		if it.page == nil {
			if it.nextPID > it.total {
				it.finished = true
				return false
			}
			pg, err := it.table.bufMgr.GetPage(it.table.Schema, it.table.Name, it.nextPID)
			if err != nil {
				it.err = err
				return false
			}
			it.curPID = it.nextPID
			it.nextPID++
			if !pageInitialized(pg) {
				it.table.bufMgr.Unpin(it.table.Schema, it.table.Name, it.curPID)
				continue
			}
			it.page = pg
			slots, _, _ := readHeader(pg)
			it.slots = slots
			it.slotIdx = 0
		}

		for it.slotIdx < it.slots {
			slot := getSlot(it.page, it.slotIdx)
			it.slotIdx++
			if slot.Length == 0 {
				continue
			}
			payload := it.page.Data[slot.Offset : slot.Offset+slot.Length]
			k, v, err := decodeTuple(payload)
			if err != nil {
				it.err = err
				return false
			}
			if !withinRange(k, it.start, it.end) {
				continue
			}
			it.key = append([]byte(nil), k...)
			it.value = append([]byte(nil), v...)
			return true
		}

		it.releasePage()
	}
}

func (it *iterator) Key() []byte {
	return it.key
}

func (it *iterator) Value() []byte {
	return it.value
}

func (it *iterator) Err() error {
	return it.err
}

func (it *iterator) Close() error {
	it.releasePage()
	it.finished = true
	return nil
}

func (it *iterator) releasePage() {
	if it.page != nil {
		it.table.bufMgr.Unpin(it.table.Schema, it.table.Name, it.curPID)
		it.page = nil
	}
}

func withinRange(key, start, end []byte) bool {
	if len(start) > 0 && bytes.Compare(key, start) < 0 {
		return false
	}
	if len(end) > 0 && bytes.Compare(key, end) >= 0 {
		return false
	}
	return true
}
