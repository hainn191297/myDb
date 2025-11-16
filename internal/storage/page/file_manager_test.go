package page

import (
	"path/filepath"
	"testing"
)

func TestFileManager_WriteRead(t *testing.T) {
	// create temp dir
	base := t.TempDir()

	// we write file: tempdir/main.db
	filePath := filepath.Join(base, "main.db")

	fm, err := NewFileManager(filePath)
	if err != nil {
		t.Fatalf("failed to create FileManager: %v", err)
	}
	defer fm.Close()

	id, _ := fm.AllocatePage()

	p := NewPage("main.db", id)
	copy(p.Data, []byte("Hello"))

	if err := fm.WritePage(p); err != nil {
		t.Fatalf("failed to write page: %v", err)
	}

	// ensure data persisted
	fm.Sync()

	readP, err := fm.ReadPage(id)
	if err != nil {
		t.Fatalf("failed to read page: %v", err)
	}

	if string(readP.Data[:5]) != "Hello" {
		t.Fatalf("data mismatch: expected 'Hello', got '%s'", string(readP.Data[:5]))
	}
}
