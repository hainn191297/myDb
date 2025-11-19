package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func TestManagerRecoverReplaysLog(t *testing.T) {
	base := t.TempDir()
	mgr := NewManager(base)

	const (
		schema = "public"
		table  = "users"
	)

	logger, err := mgr.Open(schema, table)
	if err != nil {
		t.Fatalf("open logger: %v", err)
	}

	pg := page.NewPage(1)
	copy(pg.Data, []byte("hello wal recovery"))
	if err := logger.Append(42, pg.ID, pg.Data); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := mgr.Close(); err != nil {
		t.Fatalf("close manager: %v", err)
	}

	// Run recovery to apply the WAL record.
	mgr = NewManager(base)
	if err := mgr.Recover(); err != nil {
		t.Fatalf("recover: %v", err)
	}

	walPath := filepath.Join(base, schema, table+".wal")
	if _, err := os.Stat(walPath); !os.IsNotExist(err) {
		t.Fatalf("wal file still exists after recovery: %v", err)
	}

	dataPath := filepath.Join(base, schema, table+".db")
	fm, err := page.NewFileManager(dataPath)
	if err != nil {
		t.Fatalf("open data file: %v", err)
	}
	defer fm.Close()

	got, err := fm.ReadPage(1)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !bytes.Equal(got.Data, pg.Data) {
		t.Fatalf("recovered page mismatch")
	}
}

func TestManagerRecoverTruncatedWal(t *testing.T) {
	base := t.TempDir()
	const (
		schema = "public"
		table  = "broken"
	)

	dir := filepath.Join(base, schema)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	walPath := filepath.Join(dir, table+".wal")

	// Write a partial record that should trigger an error.
	if err := os.WriteFile(walPath, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatalf("write wal: %v", err)
	}

	mgr := NewManager(base)
	if err := mgr.Recover(); err == nil {
		t.Fatalf("expected recover error for truncated wal")
	}

	if _, err := os.Stat(walPath); err != nil {
		t.Fatalf("wal file removed after failed recovery: %v", err)
	}
}
