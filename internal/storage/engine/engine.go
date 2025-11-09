package engine

import "context"

// Engine describes the pluggable storage interface referenced in the PlantUML diagram.
type Engine interface {
    Get(ctx context.Context, key []byte) ([]byte, error)
    Put(ctx context.Context, key, value []byte) error
    Delete(ctx context.Context, key []byte) error
    Scan(ctx context.Context, start, end []byte) (Iterator, error)
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

type errPlaceholder string

func (e errPlaceholder) Error() string { return string(e) }
