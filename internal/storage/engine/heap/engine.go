package heap

import (
	"bytes"
	"context"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
HeapEngine implements engine.Engine on top of heap-organized pages.

Simple key-value semantics:

  - Put = insert or replace
  - Get = linear scan
  - Delete = linear scan then mark slot free
  - Scan = full table scan with range filtering

This is O(n) for all operations, but enough for MVP.
*/
type HeapEngine struct {
	schema string
	table  string

	tm *engine.TableManager
	bm *engine.BufferManager
}

// NewHeapEngine binds heap engine to (schema, table)
func NewHeapEngine(schema, table string, tm *engine.TableManager, bm *engine.BufferManager) *HeapEngine {
	return &HeapEngine{
		schema: schema,
		table:  table,
		tm:     tm,
		bm:     bm,
	}
}

/*
Put performs insert or replacement:

  - Scan table for existing key → if found: delete old tuple
  - Insert new tuple at end
*/
func (h *HeapEngine) Put(ctx context.Context, key, value []byte) error {
	t := NewTable(h.schema, h.table, h.tm, h.bm)

	oldPID, oldSlot, _, found, err := h.locateKey(ctx, t, key, false)
	if err != nil {
		return err
	}

	if found {
		if err := t.Delete(oldPID, oldSlot); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	tuple := encodeTuple(key, value)
	_, _, ierr := t.Insert(tuple)
	return ierr
}

/*
Get performs a full scan until key is found.
*/
func (h *HeapEngine) Get(ctx context.Context, key []byte) ([]byte, error) {
	t := NewTable(h.schema, h.table, h.tm, h.bm)

	_, _, value, found, err := h.locateKey(ctx, t, key, true)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, engine.ErrKeyNotFound
	}

	return value, nil
}

/*
Delete: linear scan to locate key, then call table.Delete(pid, slot).
*/
func (h *HeapEngine) Delete(ctx context.Context, key []byte) error {
	t := NewTable(h.schema, h.table, h.tm, h.bm)

	pid, slot, _, found, err := h.locateKey(ctx, t, key, false)
	if err != nil {
		return err
	}

	if found {
		return t.Delete(pid, slot)
	}

	return nil
}

/*
Scan(start, end) performs full table scan with optional range filtering.

If start=nil and end=nil => full scan.
If start!=nil => k >= start
If end!=nil   => k < end
*/
func (h *HeapEngine) Scan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	t := NewTable(h.schema, h.table, h.tm, h.bm)

	var kvs []kvPair
	var ctxErr error

	err := t.Scan(func(pid page.PageID, slot int, off uint16, pg *page.Page) bool {
		if ctxErr = ctx.Err(); ctxErr != nil {
			return false
		}

		k, v, derr := decodeTuple(pg.Data[off:])
		if derr != nil {
			return true // skip bad tuples
		}

		if start != nil && bytes.Compare(k, start) < 0 {
			return true // k < start → skip
		}
		if end != nil && bytes.Compare(k, end) >= 0 {
			return true // k >= end → skip
		}

		kvs = append(kvs, kvPair{key: k, value: v})
		return true
	})
	if err != nil {
		return nil, err
	}
	if ctxErr != nil {
		return nil, ctxErr
	}

	return &heapIterator{kvs: kvs}, nil
}

// locateKey runs a full scan to find the tuple for key and optionally returns its value.
func (h *HeapEngine) locateKey(ctx context.Context, t *Table, key []byte, needValue bool) (page.PageID, int, []byte, bool, error) {
	var (
		found  bool
		pid    page.PageID
		slot   int
		value  []byte
		ctxErr error
	)

	err := t.Scan(func(p page.PageID, s int, off uint16, pg *page.Page) bool {
		if ctxErr = ctx.Err(); ctxErr != nil {
			return false
		}

		k, v, derr := decodeTuple(pg.Data[off:])
		if derr != nil {
			return true
		}
		if bytes.Equal(k, key) {
			found = true
			pid = p
			slot = s
			if needValue {
				value = v
			}
			return false
		}
		return true
	})

	if err != nil {
		return 0, 0, nil, false, err
	}
	if ctxErr != nil {
		return 0, 0, nil, false, ctxErr
	}

	return pid, slot, value, found, nil
}

// kvPair stores one materialized tuple for iterator consumption.
type kvPair struct {
	key   []byte
	value []byte
}

// heapIterator satisfies engine.Iterator over a materialized slice.
type heapIterator struct {
	kvs    []kvPair
	idx    int
	closed bool
}

func (it *heapIterator) Next() bool {
	if it.closed {
		return false
	}
	if it.idx >= len(it.kvs) {
		return false
	}
	it.idx++
	return true
}

func (it *heapIterator) Key() []byte {
	if it.idx == 0 || it.idx > len(it.kvs) {
		return nil
	}
	return it.kvs[it.idx-1].key
}

func (it *heapIterator) Value() []byte {
	if it.idx == 0 || it.idx > len(it.kvs) {
		return nil
	}
	return it.kvs[it.idx-1].value
}

func (it *heapIterator) Err() error {
	return nil
}

func (it *heapIterator) Close() error {
	it.closed = true
	it.kvs = nil
	return nil
}
