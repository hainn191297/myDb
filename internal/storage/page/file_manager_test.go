package page

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestFileManagerAllocateWriteRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "table.db")

	fm, err := NewFileManager(path)
	if err != nil {
		t.Fatalf("NewFileManager: %v", err)
	}
	defer fm.Close()

	pid, err := fm.AllocatePage()
	if err != nil {
		t.Fatalf("AllocatePage: %v", err)
	}
	if pid != 1 {
		t.Fatalf("expected first page ID = 1, got %d", pid)
	}

	page := NewPage(pid)
	payload := []byte("hello file manager")
	copy(page.Data, payload)

	if err := fm.WritePage(page); err != nil {
		t.Fatalf("WritePage: %v", err)
	}

	got, err := fm.ReadPage(pid)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}

	if !bytes.Equal(got.Data[:len(payload)], payload) {
		t.Fatalf("read payload mismatch: got %q want %q", got.Data[:len(payload)], payload)
	}
}

func TestFileManagerNumPagesTracksWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "counts.db")

	fm, err := NewFileManager(path)
	if err != nil {
		t.Fatalf("NewFileManager: %v", err)
	}
	defer fm.Close()

	for i := 1; i <= 3; i++ {
		pid, err := fm.AllocatePage()
		if err != nil {
			t.Fatalf("AllocatePage #%d: %v", i, err)
		}
		if int(pid) != i {
			t.Fatalf("expected pid %d got %d", i, pid)
		}

		pg := NewPage(pid)
		for j := range pg.Data {
			pg.Data[j] = byte(i)
		}
		if err := fm.WritePage(pg); err != nil {
			t.Fatalf("WritePage #%d: %v", i, err)
		}

		num, err := fm.NumPages()
		if err != nil {
			t.Fatalf("NumPages after #%d: %v", i, err)
		}
		if num != int64(i) {
			t.Fatalf("NumPages mismatch after #%d: got %d want %d", i, num, i)
		}
	}
}
