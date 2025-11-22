package btree

import (
	"context"

	"github.com/hainn191297/myDb/internal/storage/btree/mem"
	"github.com/hainn191297/myDb/internal/storage/engine"
)

// MemEngine adapts the in-memory B+ Tree to the IndexEngine interface.
// This is used for testing and development until the on-disk engine is ready.
type MemEngine struct {
	tree *mem.BTree
}

// NewMemEngine creates a new in-memory index engine.
func NewMemEngine() *MemEngine {
	return &MemEngine{
		tree: mem.New(),
	}
}

func (m *MemEngine) Insert(key, value []byte) error {
	return m.tree.Insert(key, value)
}

func (m *MemEngine) Search(key []byte) ([]byte, bool, error) {
	return m.tree.Search(key)
}

func (m *MemEngine) Delete(key []byte) error {
	return m.tree.Delete(key)
}

func (m *MemEngine) RangeScan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	// mem.RangeScan returns mem.Iterator.
	// Since the underlying implementation satisfies engine.Iterator,
	// we can return it directly (Go interface assignment).
	iter, err := m.tree.RangeScan(ctx, start, end)
	if err != nil {
		return nil, err
	}
	// We need to ensure the returned iterator is compatible.
	// Since mem.Iterator and engine.Iterator have identical methods,
	// and the underlying struct implements them, this assignment is valid
	// if the underlying struct is exported or if we just return the interface value.
	// However, we can't simply return `iter` if the return type is different interface.
	// We have to rely on the underlying type implementing the target interface.

	// Type assertion check (runtime)
	if eIter, ok := iter.(engine.Iterator); ok {
		return eIter, nil
	}
	// If for some reason it doesn't match (shouldn't happen if signatures match),
	// we might need a wrapper. But here signatures match exactly.
	return iter.(engine.Iterator), nil
}
