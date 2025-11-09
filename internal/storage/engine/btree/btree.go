package btree

import (
    "context"

    "github.com/hainn191297/myDb/internal/storage/engine"
)

// Engine is a placeholder for the future B+Tree implementation.
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
