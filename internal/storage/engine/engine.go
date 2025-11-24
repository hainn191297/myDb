package engine

import (
	"context"

	dberrors "github.com/hainn191297/myDb/internal/errors"
)

// Engine describes the pluggable storage interface.
type Engine interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Put(ctx context.Context, key, value []byte) error
	Delete(ctx context.Context, key []byte) error
	Scan(ctx context.Context, start, end []byte) (Iterator, error)
}

// IndexEngine describes the interface for index storage (e.g. B+ Tree).
type IndexEngine interface {
	Insert(key, value []byte) error
	Search(key []byte) ([]byte, bool, error)
	Delete(key []byte) error
	RangeScan(ctx context.Context, start, end []byte) (Iterator, error)
}

// Iterator streams key/value pairs in order.
type Iterator interface {
	Next() bool
	Key() []byte
	Value() []byte
	Err() error
	Close() error
}

// ErrNotImplemented is returned by stub engines until functionality is filled in.
var ErrNotImplemented = errPlaceholder("storage engine not implemented")

// ErrKeyNotFound signals that the requested key does not exist.
// Deprecated: Use dberrors.ErrKeyNotFound instead.
var ErrKeyNotFound = dberrors.ErrKeyNotFound

type errPlaceholder string

func (e errPlaceholder) Error() string { return string(e) }
