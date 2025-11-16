package engine

import (
	"testing"

	"github.com/hainn191297/myDb/internal/storage/page"
)

func TestBufferManagerProvidesPerTablePool(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)
	bm := NewBufferManager(tm, nil, 2)

	fm, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open table: %v", err)
	}
	id, _ := fm.AllocatePage()
	pg := page.NewPage("users", id)
	copy(pg.Data, []byte("user"))
	if err := fm.WritePage(pg); err != nil {
		t.Fatalf("write page: %v", err)
	}

	got, err := bm.GetPage("public", "users", id)
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	if string(got.Data[:4]) != "user" {
		t.Fatalf("expected 'user', got %q", got.Data[:4])
	}

	if len(bm.pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(bm.pools))
	}
}

func TestBufferManagerCreatesDistinctPoolsPerTable(t *testing.T) {
	dir := t.TempDir()
	tm := NewTableManager(dir)
	bm := NewBufferManager(tm, nil, 1)

	fmUsers, err := tm.OpenTable("public", "users")
	if err != nil {
		t.Fatalf("open users: %v", err)
	}
	uid, _ := fmUsers.AllocatePage()
	upg := page.NewPage("users", uid)
	copy(upg.Data, []byte("U000"))
	fmUsers.WritePage(upg)

	fmOrders, err := tm.OpenTable("sales", "orders")
	if err != nil {
		t.Fatalf("open orders: %v", err)
	}
	oid, _ := fmOrders.AllocatePage()
	opg := page.NewPage("orders", oid)
	copy(opg.Data, []byte("O000"))
	fmOrders.WritePage(opg)

	if _, err := bm.GetPage("public", "users", uid); err != nil {
		t.Fatalf("Get users: %v", err)
	}
	if _, err := bm.GetPage("sales", "orders", oid); err != nil {
		t.Fatalf("Get orders: %v", err)
	}

	if len(bm.pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(bm.pools))
	}
}
