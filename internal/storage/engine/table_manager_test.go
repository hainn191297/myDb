package engine

import (
	"testing"
)

func TestTableManagerOpenTableCachesFileManager(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)

	fm1, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open table: %v", err)
	}
	fm2, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open table second: %v", err)
	}

	if fm1 != fm2 {
		t.Fatalf("expected cached FileManager, got different instances")
	}
}

func TestTableManagerCloseClosesAll(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)

	if _, err := tm.OpenTable("public", "users"); err != nil {
		t.Fatalf("open table: %v", err)
	}
	if _, err := tm.OpenTable("public", "orders"); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := tm.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
}
