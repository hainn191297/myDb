package wal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func TestFileLoggerAppendAndSync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	logger, err := OpenLogger(path)
	if err != nil {
		t.Fatalf("open logger: %v", err)
	}
	defer logger.Close()

	if err := logger.Append("public.users", page.PageID(1), []byte("data")); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wal: %v", err)
	}
	// table len (2 bytes) + table bytes (12) + pageID (8) + data len (4) + data (4)
	expectedLen := 2 + len("public.users") + 8 + 4 + len("data")
	if len(content) != expectedLen {
		t.Fatalf("unexpected record length: got %d want %d", len(content), expectedLen)
	}
}
