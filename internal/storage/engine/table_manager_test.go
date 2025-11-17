package engine

import (
	"testing"
)

func TestTableManagerOpenTableCachesFileManager(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)

	fid1, fm1, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open table: %v", err)
	}
	fid2, fm2, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open table second: %v", err)
	}

	if fid1 != fid2 {
		t.Fatalf("expected same FileID, got %d vs %d", fid1, fid2)
	}
	if fm1 != fm2 {
		t.Fatalf("expected cached FileManager, got different instances")
	}
}

func TestTableManagerAssignsUniqueIDsAndLookup(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)

	fidUsers, _, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open users: %v", err)
	}
	fidOrders, _, err := tm.OpenTable("public", "orders")
	if err != nil {
		t.Fatalf("open orders: %v", err)
	}
	if fidUsers == fidOrders {
		t.Fatalf("expected different FileIDs, got %d", fidUsers)
	}

	if fm, ok := tm.LookupByFileID(fidUsers); !ok || fm == nil {
		t.Fatalf("LookupByFileID(%d) failed", fidUsers)
	}
}

func TestTableManagerCloseClosesAll(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)

	if _, _, err := tm.OpenTable("public", "users"); err != nil {
		t.Fatalf("open table: %v", err)
	}
	if _, _, err := tm.OpenTable("sales", "orders"); err != nil {
		t.Fatalf("open table: %v", err)
	}

	if err := tm.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
}
