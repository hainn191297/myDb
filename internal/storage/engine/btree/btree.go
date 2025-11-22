// Package btree defines the on-disk B+Tree storage engine slot. It currently
// holds stubs until the page-backed implementation is filled in.
package btree

import (
	"context"

	"github.com/hainn191297/myDb/internal/storage/engine"
)

// Engine is a placeholder for the future B+Tree implementation.
//
// NOTE: This is currently a stub. The actual on-disk B+ Tree implementation
// is pending. For now, see internal/storage/btree/mem for an in-memory
// playground implementation of the algorithms.
type Engine struct{}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Get(ctx context.Context, key []byte) ([]byte, error) {
	return nil, engine.ErrNotImplemented
}

func (e *Engine) Put(ctx context.Context, key, value []byte) error {
	return engine.ErrNotImplemented
}

func (e *Engine) Delete(ctx context.Context, key []byte) error {
	return engine.ErrNotImplemented
}

func (e *Engine) Scan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	return nil, engine.ErrNotImplemented
}
