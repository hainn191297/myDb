package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
TableManager handles:

- schema directory creation
- mapping (schema.table) → FileManager
- assigning FileID (unique int for global buffer pool)
*/
type TableManager struct {
	basePath string

	mu       sync.Mutex
	nextFID  uint32 // next FileID
	byName   map[string]*page.FileManager
	byFileID map[uint32]*page.FileManager
	names    map[uint32]string // fid -> "schema.table"
}

func NewTableManager(basePath string) *TableManager {
	return &TableManager{
		basePath: basePath,
		nextFID:  1,
		byName:   make(map[string]*page.FileManager),
		byFileID: make(map[uint32]*page.FileManager),
		names:    make(map[uint32]string),
	}
}

// CreateSchema ensures "basePath/schema/" exists.
func (tm *TableManager) CreateSchema(schema string) error {
	return os.MkdirAll(filepath.Join(tm.basePath, schema), 0o755)
}

// tableKey → "schema.table"
func tableKey(schema, table string) string {
	return schema + "." + table
}

/*
OpenTable returns:

- FileID (uint32)
- *FileManager

Caches FileManager in memory so multiple engines or scans share the same FD.
*/
func (tm *TableManager) OpenTable(schema, table string) (uint32, *page.FileManager, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := tableKey(schema, table)

	// cached?
	if fm, ok := tm.byName[key]; ok {
		// find FileID from byFileID
		for fid, ref := range tm.byFileID {
			if ref == fm {
				return fid, fm, nil
			}
		}
		// Should not happen
		return 0, nil, fmt.Errorf("tm: fileID missing for table %s", key)
	}

	// ensure schema directory exists
	dir := filepath.Join(tm.basePath, schema)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, nil, err
	}

	// open file
	path := filepath.Join(dir, table+".db")
	fm, err := page.NewFileManager(path)
	if err != nil {
		return 0, nil, err
	}

	// assign new FileID
	fid := tm.nextFID
	tm.nextFID++

	// cache
	tm.byName[key] = fm
	tm.byFileID[fid] = fm
	tm.names[fid] = key

	return fid, fm, nil
}

// LookupByFileID returns FileManager by FileID (used by BufferManager flush)
func (tm *TableManager) LookupByFileID(fid uint32) (*page.FileManager, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	fm, ok := tm.byFileID[fid]
	return fm, ok
}

// LookupName returns (schema, table) for a given FileID.
func (tm *TableManager) LookupName(fid uint32) (string, string, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key, ok := tm.names[fid]
	if !ok {
		return "", "", false
	}
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// Close closes every FileManager
func (tm *TableManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var lastErr error
	for fid, fm := range tm.byFileID {
		if err := fm.Close(); err != nil {
			lastErr = err
		}
		delete(tm.byFileID, fid)
		delete(tm.names, fid)
	}

	for key := range tm.byName {
		delete(tm.byName, key)
	}

	return lastErr
}
