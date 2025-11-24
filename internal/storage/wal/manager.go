package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hainn191297/myDb/internal/storage/page"
)

/*
Manager caches WAL loggers for (schema, table) combos.
Each log lives alongside its table data file: basePath/schema/table.wal.
*/
type Manager struct {
	basePath string

	mu    sync.Mutex
	logs  map[string]Logger
	paths map[string]string
}

func NewManager(basePath string) *Manager {
	return &Manager{
		basePath: basePath,
		logs:     make(map[string]Logger),
		paths:    make(map[string]string),
	}
}

func (m *Manager) walKey(schema, table string) string {
	return schema + "." + table
}

// Open returns a cached Logger for schema.table, creating it lazily.
func (m *Manager) Open(schema, table string) (Logger, error) {
	key := m.walKey(schema, table)

	m.mu.Lock()
	defer m.mu.Unlock()

	if logger, ok := m.logs[key]; ok {
		return logger, nil
	}

	dir := filepath.Join(m.basePath, schema)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, table+".wal")
	logger, err := OpenLogger(path)
	if err != nil {
		return nil, err
	}

	m.logs[key] = logger
	m.paths[key] = path
	return logger, nil
}

// Close releases every cached logger.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for key, logger := range m.logs {
		if err := logger.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
		delete(m.logs, key)
		delete(m.paths, key)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// SyncAll flushes every open WAL logger.
func (m *Manager) SyncAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for key, logger := range m.logs {
		if err := logger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Recover scans all WAL files under basePath and replays them into their data files.
// After successful replay, the WAL file is removed to avoid double application.
func (m *Manager) Recover() error {
	files, err := m.listWalFiles()
	if err != nil {
		return err
	}
	for _, wf := range files {
		if err := m.replayWalFile(wf); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) listWalFiles() ([]walFile, error) {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("wal: read base dir: %w", err)
	}

	var files []walFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		schema := entry.Name()
		schemaPath := filepath.Join(m.basePath, schema)
		children, err := os.ReadDir(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("wal: read schema dir %s: %w", schemaPath, err)
		}
		for _, child := range children {
			if child.IsDir() {
				continue
			}
			name := child.Name()
			if !strings.HasSuffix(name, ".wal") {
				continue
			}
			table := strings.TrimSuffix(name, ".wal")
			files = append(files, walFile{
				schema: schema,
				table:  table,
				path:   filepath.Join(schemaPath, name),
			})
		}
	}
	return files, nil
}

func (m *Manager) replayWalFile(info walFile) error {
	key := m.walKey(info.schema, info.table)
	m.mu.Lock()
	if logger, ok := m.logs[key]; ok {
		logger.Close()
		delete(m.logs, key)
	}
	delete(m.paths, key)
	m.mu.Unlock()

	f, err := os.Open(info.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("wal: open log %s: %w", info.path, err)
	}
	closed := false
	defer func() {
		if !closed {
			f.Close()
		}
	}()

	reader := bufio.NewReader(f)
	header := make([]byte, 16) // fileID + pageID + dataLen
	dataBuf := make([]byte, page.PageSize)
	dataPath := filepath.Join(m.basePath, info.schema, info.table+".db")

	fm, err := page.NewFileManager(dataPath)
	if err != nil {
		return fmt.Errorf("wal: open data file %s: %w", dataPath, err)
	}
	defer fm.Close()

	applied := false
	for {
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return fmt.Errorf("wal: truncated header in %s", info.path)
			}
			return fmt.Errorf("wal: read header from %s: %w", info.path, err)
		}

		fileID := binary.LittleEndian.Uint32(header[0:4])
		pid := page.PageID(int64(binary.LittleEndian.Uint64(header[4:12])))
		dataLen := binary.LittleEndian.Uint32(header[12:16])
		if dataLen == 0 {
			return fmt.Errorf("wal: empty payload for file %d page %d in %s", fileID, pid, info.path)
		}
		if int(dataLen) > len(dataBuf) {
			dataBuf = make([]byte, dataLen)
		}
		if _, err := io.ReadFull(reader, dataBuf[:dataLen]); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return fmt.Errorf("wal: truncated payload in %s", info.path)
			}
			return fmt.Errorf("wal: read payload from %s: %w", info.path, err)
		}
		if dataLen != uint32(page.PageSize) {
			return fmt.Errorf("wal: invalid payload size %d in %s (expected %d)", dataLen, info.path, page.PageSize)
		}

		pg := &page.Page{ID: pid, Data: dataBuf[:dataLen]}
		if err := fm.WritePage(pg); err != nil {
			return fmt.Errorf("wal: write %s.%s page %d: %w", info.schema, info.table, pid, err)
		}
		applied = true
	}

	if applied {
		if err := fm.Sync(); err != nil {
			return fmt.Errorf("wal: sync %s.%s: %w", info.schema, info.table, err)
		}
	}

	// Close log file before removing on Windows systems.
	if err := f.Close(); err != nil {
		return fmt.Errorf("wal: close log %s: %w", info.path, err)
	}
	closed = true

	if err := os.Remove(info.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("wal: remove %s: %w", info.path, err)
	}

	return nil
}

type walFile struct {
	schema string
	table  string
	path   string
}
