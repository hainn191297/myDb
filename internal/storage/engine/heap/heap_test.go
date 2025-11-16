package heap

import (
	"testing"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

func TestTableInsertAndScan(t *testing.T) {
	dir := t.TempDir()

	tm := engine.NewTableManager(dir)
	wm := wal.NewManager(dir)
	bm := engine.NewBufferManager(tm, wm, 4)

	table := NewTable("public", "users", bm)

	values := []string{"alice", "bob", "carol"}
	for _, v := range values {
		if _, err := table.Insert([]byte(v)); err != nil {
			t.Fatalf("insert %s: %v", v, err)
		}
	}

	got := make(map[string]bool)
	if err := table.Scan(func(_ RecordID, data []byte) bool {
		got[string(data)] = true
		return true
	}); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if len(got) != len(values) {
		t.Fatalf("expected %d values, got %d", len(values), len(got))
	}

	for _, v := range values {
		if !got[v] {
			t.Fatalf("missing value %s", v)
		}
	}
}

func TestTableDeleteReusesSlot(t *testing.T) {
	dir := t.TempDir()
	tm := engine.NewTableManager(dir)
	wm := wal.NewManager(dir)
	bm := engine.NewBufferManager(tm, wm, 2)

	table := NewTable("public", "users", bm)

	rid1, err := table.Insert([]byte("foo"))
	if err != nil {
		t.Fatalf("insert foo: %v", err)
	}
	if _, err := table.Insert([]byte("bar")); err != nil {
		t.Fatalf("insert bar: %v", err)
	}

	if err := table.Delete(rid1); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rid3, err := table.Insert([]byte("baz"))
	if err != nil {
		t.Fatalf("insert baz: %v", err)
	}

	if rid3.Slot != rid1.Slot {
		t.Fatalf("expected slot reuse %d, got %d", rid1.Slot, rid3.Slot)
	}

	values := map[string]bool{}
	if err := table.Scan(func(_ RecordID, data []byte) bool {
		values[string(data)] = true
		return true
	}); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if values["foo"] {
		t.Fatalf("deleted tuple still visible")
	}
	if !values["bar"] || !values["baz"] {
		t.Fatalf("missing tuples: %v", values)
	}
}
