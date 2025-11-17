package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
Logger captures redo information before dirty pages flush to disk.
Append order must be persisted (via Sync) before the corresponding
data pages are written for durability.

New WAL format:

[ fileID   : uint32 ]
[ pageID   : int64  ]
[ dataLen  : uint32 ]
[ data     : []byte ]
*/
type Logger interface {
	Append(fileID uint32, pid page.PageID, data []byte) error
	Sync() error
	Close() error
}

type fileLogger struct {
	mu   sync.Mutex
	file *os.File
}

// OpenLogger opens (or creates) a WAL file at the given path.
func OpenLogger(path string) (Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("wal: open %s: %w", path, err)
	}
	return &fileLogger{file: f}, nil
}

func (l *fileLogger) Append(fileID uint32, pid page.PageID, data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Preallocate buffer
	buf := make([]byte, 4+8+4+len(data))
	offset := 0

	// fileID
	binary.LittleEndian.PutUint32(buf[offset:], fileID)
	offset += 4

	// pageID
	binary.LittleEndian.PutUint64(buf[offset:], uint64(pid))
	offset += 8

	// dataLen
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(data)))
	offset += 4

	// data
	copy(buf[offset:], data)

	if _, err := l.file.Write(buf); err != nil {
		return fmt.Errorf("wal: append failed: %w", err)
	}
	return nil
}

func (l *fileLogger) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("wal: sync failed: %w", err)
	}
	return nil
}

func (l *fileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
