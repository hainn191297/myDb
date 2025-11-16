package wal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
