package wal

import (
	"encoding/binary"
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

	const fileID = uint32(123)
	payload := []byte("data")

	if err := logger.Append(fileID, page.PageID(1), payload); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read wal: %v", err)
	}

	expectedLen := 4 + 8 + 4 + len(payload)
	if len(content) != expectedLen {
		t.Fatalf("unexpected record length: got %d want %d", len(content), expectedLen)
	}

	gotFileID := binary.LittleEndian.Uint32(content[0:4])
	if gotFileID != fileID {
		t.Fatalf("fileID mismatch: got %d want %d", gotFileID, fileID)
	}
	gotPageID := binary.LittleEndian.Uint64(content[4:12])
	if gotPageID != uint64(1) {
		t.Fatalf("pageID mismatch: got %d want %d", gotPageID, 1)
	}
	gotLen := binary.LittleEndian.Uint32(content[12:16])
	if int(gotLen) != len(payload) {
		t.Fatalf("payload len mismatch: got %d want %d", gotLen, len(payload))
	}
	if string(content[16:]) != string(payload) {
		t.Fatalf("payload mismatch: got %q want %q", content[16:], payload)
	}
}
