package page

import (
	"os"
	"testing"
)

func TestFileManager_WriteRead(t *testing.T) {
	dir := "temp"

	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	fm, err := NewFileManager(dir)
	if err != nil {
		t.Fatalf("failed to create FileManager: %v", err)
	}

	defer fm.Close()

	id, _ := fm.AllocatePage()

	p := NewPage(id)
	copy(p.Data, []byte("Hello"))

	if err := fm.WritePage(p); err != nil {
		t.Fatalf("failed to write page: %v", err)
	}

	// write disk
	fm.Sync()

	readP, err := fm.ReadPage(id)
	if err != nil {
		t.Fatalf("failed to read page: %v", err)
	}

	if string(readP.Data[:5]) != "Hello" {
		t.Fatalf("data mismatch: expected 'Hello', got '%s'", string(readP.Data[:5]))
	}
}
