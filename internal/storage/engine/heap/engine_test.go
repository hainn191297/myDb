package heap

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/storage/wal"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	tm := engine.NewTableManager(dir)
	wm := wal.NewManager(dir)
	bm := engine.NewBufferManager(tm, wm, 4)
	return NewEngine("public", "kv", bm)
}

func TestEnginePutGetDelete(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()

	if err := e.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := e.Put(ctx, []byte("k2"), []byte("v2")); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Overwrite k1
	if err := e.Put(ctx, []byte("k1"), []byte("v3")); err != nil {
		t.Fatalf("put overwrite: %v", err)
	}

	v, err := e.Get(ctx, []byte("k1"))
	if err != nil || string(v) != "v3" {
		t.Fatalf("get k1 after overwrite: %v val=%s", err, string(v))
	}

	if err := e.Delete(ctx, []byte("k1")); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := e.Get(ctx, []byte("k1")); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestEngineScan(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()
	keys := []string{"b", "a", "c"}
	for _, k := range keys {
		if err := e.Put(ctx, []byte(k), []byte(strings.ToUpper(k))); err != nil {
			t.Fatalf("put %s: %v", k, err)
		}
	}

	it, err := e.Scan(ctx, nil, nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	defer it.Close()

	var seen []string
	for it.Next() {
		seen = append(seen, string(it.Key()))
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iterator err: %v", err)
	}
	if !reflect.DeepEqual(seen, []string{"b", "a", "c"}) {
		t.Fatalf("unexpected iteration order: %v", seen)
	}
}
