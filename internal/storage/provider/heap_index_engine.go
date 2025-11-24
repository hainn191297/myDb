package provider

import (
	"bytes"
	"context"
	"sort"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/engine/heap"
)

// heapIndexEngine is a durable index engine backed by heap pages.
// It trades performance for simplicity: range scans materialize and sort in-memory.
type heapIndexEngine struct {
	heap *heap.HeapEngine
}

type kv struct {
	k []byte
	v []byte
}

func newHeapIndexEngine(schema, name string, tm *engine.TableManager, bm *engine.BufferManager) *heapIndexEngine {
	// Indexes don't need PK optimization, so pass nil for tableDef and provider
	return &heapIndexEngine{
		heap: heap.NewHeapEngine(schema, name, tm, bm, nil, nil),
	}
}

func (h *heapIndexEngine) Insert(key, value []byte) error {
	return h.heap.Put(context.Background(), key, value)
}

func (h *heapIndexEngine) Search(key []byte) ([]byte, bool, error) {
	val, err := h.heap.Get(context.Background(), key)
	if err != nil {
		if err == engine.ErrKeyNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

func (h *heapIndexEngine) Delete(key []byte) error {
	return h.heap.Delete(context.Background(), key)
}

func (h *heapIndexEngine) RangeScan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	iter, err := h.heap.Scan(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var rows []kv
	for iter.Next() {
		k := append([]byte(nil), iter.Key()...)
		v := append([]byte(nil), iter.Value()...)
		if start != nil && bytes.Compare(k, start) < 0 {
			continue
		}
		if end != nil && bytes.Compare(k, end) >= 0 {
			continue
		}
		rows = append(rows, kv{k: k, v: v})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool {
		return bytes.Compare(rows[i].k, rows[j].k) < 0
	})

	return &heapIndexIterator{rows: rows}, nil
}

type heapIndexIterator struct {
	rows []kv
	idx  int
}

func (it *heapIndexIterator) Next() bool {
	if it.idx >= len(it.rows) {
		return false
	}
	it.idx++
	return true
}

func (it *heapIndexIterator) Key() []byte {
	if it.idx == 0 || it.idx > len(it.rows) {
		return nil
	}
	return it.rows[it.idx-1].k
}

func (it *heapIndexIterator) Value() []byte {
	if it.idx == 0 || it.idx > len(it.rows) {
		return nil
	}
	return it.rows[it.idx-1].v
}

func (it *heapIndexIterator) Err() error { return nil }

func (it *heapIndexIterator) Close() error {
	it.rows = nil
	return nil
}
