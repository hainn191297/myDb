package heap

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/hainn191297/myDb/internal/schema"
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

	// Primary key index support
	tableDef  *schema.TableDef
	pkColumns []string
	provider  StorageProvider // Interface to access index engines
}

// StorageProvider is a minimal interface for accessing index engines.
// This avoids circular dependency with the full provider package.
type StorageProvider interface {
	Index(schema, table, indexName string) (engine.IndexEngine, error)
}

// NewHeapEngine binds heap engine to (schema, table)
func NewHeapEngine(
	schema, table string,
	tm *engine.TableManager,
	bm *engine.BufferManager,
	tableDef *schema.TableDef,
	provider StorageProvider,
) *HeapEngine {
	// Extract primary key columns
	pkCols := extractPKColumns(tableDef)

	return &HeapEngine{
		schema:    schema,
		table:     table,
		tm:        tm,
		bm:        bm,
		tableDef:  tableDef,
		pkColumns: pkCols,
		provider:  provider,
	}
}

// extractPKColumns returns the list of primary key column names.
func extractPKColumns(td *schema.TableDef) []string {
	if td == nil {
		return nil
	}
	var pk []string
	for _, col := range td.Columns {
		if col.PrimaryKey {
			pk = append(pk, col.Name)
		}
	}
	return pk
}

/*
Put performs insert or replacement.
If a PK index exists, uses O(log n) lookup to check for existing key.
Otherwise, falls back to O(n) scan.
*/
func (h *HeapEngine) Put(ctx context.Context, key, value []byte) error {
	t := NewTable(h.schema, h.table, h.tm, h.bm)

	// Try PK index for fast existence check
	if idxEng, hasIndex := h.getPKIndexEngine(); hasIndex {
		oldLocation, found, err := idxEng.Search(key)
		if err != nil {
			// Index error, fall through to scan-based approach
			return h.putViaScan(ctx, t, key, value)
		}

		if found {
			// Delete old tuple at indexed location; if location encoding is missing/old, fall back to scan.
			pid, slot, err := decodeLocation(oldLocation)
			if err == nil {
				_ = t.Delete(pid, slot)
			} else {
				if pid, slot, _, located, lerr := h.locateKey(ctx, t, key, false); lerr == nil && located {
					_ = t.Delete(pid, slot)
				}
			}
		}

		// Insert new tuple
		tuple := encodeTuple(key, value)
		newPID, newSlot, err := t.Insert(tuple)
		if err != nil {
			return err
		}

		// Update index with new location
		newLocation := encodeLocation(newPID, newSlot)
		if found {
			// Update: delete old key first, then insert new
			_ = idxEng.Delete(key)
		}
		if err := idxEng.Insert(key, newLocation); err != nil {
			// Index insert failed, but data is written
			// This is acceptable; worst case is index rebuild needed
			return nil
		}

		return nil
	}

	// Fallback: O(n) scan-based Put
	return h.putViaScan(ctx, t, key, value)
}

// putViaScan is the O(n) fallback implementation.
func (h *HeapEngine) putViaScan(ctx context.Context, t *Table, key, value []byte) error {
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
Get performs a primary key lookup.
If a PK index exists, uses O(log n) index lookup.
Otherwise, falls back to O(n) full scan.
*/
func (h *HeapEngine) Get(ctx context.Context, key []byte) ([]byte, error) {
	// Try PK index first for O(log n) lookup
	if idxEng, hasIndex := h.getPKIndexEngine(); hasIndex {
		location, found, err := idxEng.Search(key)
		if err != nil {
			return nil, fmt.Errorf("heap: pk index search: %w", err)
		}
		if !found {
			return nil, engine.ErrKeyNotFound
		}

		// Decode location to (pageID, slotID); if not a location, fall back to scan.
		pid, slot, err := decodeLocation(location)
		if err != nil {
			return h.getViaScan(ctx, key)
		}

		// Direct page/slot access
		t := NewTable(h.schema, h.table, h.tm, h.bm)
		tupleData, err := t.GetTuple(ctx, pid, slot)
		if err != nil {
			// Tuple deleted/moved? Fall back to scan
			return h.getViaScan(ctx, key)
		}

		// Decode tuple to get value
		k, v, err := decodeTuple(tupleData)
		if err != nil {
			return nil, fmt.Errorf("heap: decode tuple: %w", err)
		}

		// Verify key matches (paranoid check for index corruption)
		if !bytes.Equal(k, key) {
			// Index/heap out of sync, fall back to scan
			return h.getViaScan(ctx, key)
		}

		return v, nil
	}

	// Fallback: O(n) full scan
	return h.getViaScan(ctx, key)
}

// getViaScan is the O(n) fallback using full table scan.
func (h *HeapEngine) getViaScan(ctx context.Context, key []byte) ([]byte, error) {
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
Delete removes a tuple by key.
If a PK index exists, uses O(log n) lookup to find the tuple.
Otherwise, falls back to O(n) scan.
*/
func (h *HeapEngine) Delete(ctx context.Context, key []byte) error {
	// Try PK index for fast lookup
	if idxEng, hasIndex := h.getPKIndexEngine(); hasIndex {
		location, found, err := idxEng.Search(key)
		if err != nil {
			// Index error, fall through to scan-based approach
			return h.deleteViaScan(ctx, key)
		}

		if !found {
			return nil // Already deleted or never existed
		}

		// Decode location and delete tuple; fall back to scan if not a location.
		pid, slot, err := decodeLocation(location)
		if err != nil {
			return h.deleteViaScan(ctx, key)
		}

		t := NewTable(h.schema, h.table, h.tm, h.bm)
		if err := t.Delete(pid, slot); err != nil {
			// Tuple might already be deleted, that's okay
			_ = idxEng.Delete(key)
			return nil
		}

		// Remove from index
		if err := idxEng.Delete(key); err != nil {
			// Index delete failed, but tuple is deleted
			// This is acceptable; worst case is dangling index entry
		}

		return nil
	}

	// Fallback: O(n) scan-based Delete
	return h.deleteViaScan(ctx, key)
}

// deleteViaScan is the O(n) fallback implementation.
func (h *HeapEngine) deleteViaScan(ctx context.Context, key []byte) error {
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

// encodeLocation serializes (pageID, slotID) as a 12-byte value for index storage.
func encodeLocation(pid page.PageID, slot int) []byte {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint64(buf[0:8], uint64(pid))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(slot))
	return buf
}

// decodeLocation deserializes the 12-byte index value back to (pageID, slotID).
func decodeLocation(data []byte) (page.PageID, int, error) {
	if len(data) != 12 {
		return 0, 0, fmt.Errorf("heap: invalid location encoding (expected 12 bytes, got %d)", len(data))
	}
	pid := page.PageID(binary.LittleEndian.Uint64(data[0:8]))
	slot := int(binary.LittleEndian.Uint32(data[8:12]))
	return pid, slot, nil
}

// getPKIndexEngine returns the primary key index engine if one exists.
func (h *HeapEngine) getPKIndexEngine() (engine.IndexEngine, bool) {
	if len(h.pkColumns) == 0 || h.provider == nil {
		return nil, false
	}

	// Primary key index naming convention: __pk_{table}_{column}
	// This matches the convention in executor.go:385
	indexName := fmt.Sprintf("__pk_%s_%s", h.table, h.pkColumns[0])

	idxEng, err := h.provider.Index(h.schema, h.table, indexName)
	if err != nil {
		// Index doesn't exist (e.g., table just created, index not yet built)
		return nil, false
	}

	return idxEng, true
}

// GetByLocation retrieves a tuple by its physical location (encoded pageID + slotID).
// This is used by IndexScanOp.
func (h *HeapEngine) GetByLocation(ctx context.Context, location []byte) ([]byte, error) {
	pid, slot, err := decodeLocation(location)
	if err != nil {
		return nil, fmt.Errorf("heap: decode location: %w", err)
	}

	t := NewTable(h.schema, h.table, h.tm, h.bm)
	tuple, err := t.GetTuple(ctx, pid, slot)
	if err != nil {
		return nil, err
	}

	_, value, err := decodeTuple(tuple)
	if err != nil {
		return nil, fmt.Errorf("heap: decode tuple: %w", err)
	}
	return value, nil
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
