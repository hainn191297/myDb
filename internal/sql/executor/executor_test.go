package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/engine"
	"github.com/hainn191297/myDb/internal/txn"
)

func TestExecutorTxnLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr := txn.NewManager(nil, nil)
	session := &SessionTxn{}

	beginPlan := planner.Plan{Root: &planner.TxnOp{Action: planner.TxnBegin}}
	exec := New(beginPlan, Options{TxnManager: mgr, SessionTxn: session})
	if _, err := exec.Next(ctx); err != nil {
		t.Fatalf("begin txn: %v", err)
	}
	if session.Current == nil {
		t.Fatalf("session txn not set after begin")
	}

	commitPlan := planner.Plan{Root: &planner.TxnOp{Action: planner.TxnCommit}}
	exec = New(commitPlan, Options{TxnManager: mgr, SessionTxn: session})
	if _, err := exec.Next(ctx); err != nil {
		t.Fatalf("commit txn: %v", err)
	}
	if session.Current != nil {
		t.Fatalf("session txn not cleared after commit")
	}
}

func TestExecutorTxnErrors(t *testing.T) {
	ctx := context.Background()
	mgr := txn.NewManager(nil, nil)
	session := &SessionTxn{}

	// Commit without begin
	commitPlan := planner.Plan{Root: &planner.TxnOp{Action: planner.TxnCommit}}
	exec := New(commitPlan, Options{TxnManager: mgr, SessionTxn: session})
	if _, err := exec.Next(ctx); err != ErrNoActiveTxn {
		t.Fatalf("commit without txn should error, got %v", err)
	}

	// Begin twice
	beginPlan := planner.Plan{Root: &planner.TxnOp{Action: planner.TxnBegin}}
	exec = New(beginPlan, Options{TxnManager: mgr, SessionTxn: session})
	if _, err := exec.Next(ctx); err != nil {
		t.Fatalf("first begin: %v", err)
	}
	exec = New(beginPlan, Options{TxnManager: mgr, SessionTxn: session})
	if _, err := exec.Next(ctx); err != ErrTxnAlreadyActive {
		t.Fatalf("second begin expected ErrTxnAlreadyActive, got %v", err)
	}
}

func TestExecutorSeqScanReturnsRows(t *testing.T) {
	ctx := context.Background()
	provider := &fakeProvider{
		engines: map[string]engine.Engine{
			"public.users": &fakeEngine{rows: []kv{
				{key: []byte("k1"), value: []byte("v1")},
				{key: []byte("k2"), value: []byte("v2")},
			}},
		},
	}

	plan := planner.Plan{
		Root: &planner.SeqScanOp{
			Schema:  "public",
			Table:   "users",
			Columns: []string{"key", "value"},
		},
	}
	exec := New(plan, Options{Provider: provider})
	defer exec.Close()

	ok, err := exec.Next(ctx)
	if err != nil || !ok {
		t.Fatalf("first Next: ok=%v err=%v", ok, err)
	}
	row := exec.Row()
	if len(row.Columns) != 2 || string(row.Values[0]) != "k1" || string(row.Values[1]) != "v1" {
		t.Fatalf("unexpected row %#v", row)
	}

	ok, err = exec.Next(ctx)
	if err != nil || !ok {
		t.Fatalf("second Next: ok=%v err=%v", ok, err)
	}
	row = exec.Row()
	if string(row.Values[0]) != "k2" || string(row.Values[1]) != "v2" {
		t.Fatalf("second row mismatch %#v", row)
	}

	ok, err = exec.Next(ctx)
	if err != nil {
		t.Fatalf("final Next err=%v", err)
	}
	if ok {
		t.Fatalf("expected no more rows")
	}
}

func TestExecutorSeqScanColumnValidation(t *testing.T) {
	ctx := context.Background()
	provider := &fakeProvider{
		engines: map[string]engine.Engine{
			"public.users": &fakeEngine{},
		},
	}

	plan := planner.Plan{
		Root: &planner.SeqScanOp{
			Schema:  "public",
			Table:   "users",
			Columns: []string{"unknown"},
		},
	}

	exec := New(plan, Options{Provider: provider})
	defer exec.Close()

	if _, err := exec.Next(ctx); err == nil {
		t.Fatalf("expected error for invalid column")
	}
}

type fakeProvider struct {
	engines map[string]engine.Engine
	indexes map[string]engine.IndexEngine
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		engines: make(map[string]engine.Engine),
		indexes: make(map[string]engine.IndexEngine),
	}
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{
		rows: make([]kv, 0),
	}
}

func (p *fakeProvider) Engine(schema, table string) (engine.Engine, error) {
	if eng, ok := p.engines[schema+"."+table]; ok {
		return eng, nil
	}
	return nil, errors.New("not found")
}

func (p *fakeProvider) Index(schema, table, indexName string) (engine.IndexEngine, error) {
	if p.indexes == nil {
		p.indexes = make(map[string]engine.IndexEngine)
	}
	key := schema + "." + table + "." + indexName
	if idx, ok := p.indexes[key]; ok {
		return idx, nil
	}
	// Auto-create for tests
	idx := &fakeIndexEngine{data: make(map[string][]byte)}
	p.indexes[key] = idx
	return idx, nil
}

type kv struct {
	key   []byte
	value []byte
}

type fakeEngine struct {
	rows []kv
}

func (f *fakeEngine) Get(ctx context.Context, key []byte) ([]byte, error) {
	for _, row := range f.rows {
		if string(row.key) == string(key) {
			return row.value, nil
		}
	}
	return nil, engine.ErrKeyNotFound
}

func (f *fakeEngine) Put(ctx context.Context, key, value []byte) error {
	// Simple append/replace for test
	for i, row := range f.rows {
		if string(row.key) == string(key) {
			f.rows[i].value = value
			return nil
		}
	}
	f.rows = append(f.rows, kv{key: key, value: value})
	return nil
}

func (f *fakeEngine) Delete(ctx context.Context, key []byte) error {
	for i, row := range f.rows {
		if string(row.key) == string(key) {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeEngine) Scan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	return &fakeIterator{rows: f.rows}, nil
}

type fakeIndexEngine struct {
	data map[string][]byte
}

func (f *fakeIndexEngine) Insert(key, value []byte) error {
	f.data[string(key)] = value
	return nil
}

func (f *fakeIndexEngine) Search(key []byte) ([]byte, bool, error) {
	val, ok := f.data[string(key)]
	return val, ok, nil
}

func (f *fakeIndexEngine) Delete(key []byte) error {
	delete(f.data, string(key))
	return nil
}

func (f *fakeIndexEngine) RangeScan(ctx context.Context, start, end []byte) (engine.Iterator, error) {
	// Very basic range scan for tests
	var rows []kv
	for k, v := range f.data {
		if start != nil && k < string(start) {
			continue
		}
		if end != nil && k >= string(end) { // Assuming end is exclusive or handled by caller
			continue
		}
		rows = append(rows, kv{key: []byte(k), value: v})
	}
	return &fakeIterator{rows: rows}, nil
}

type fakeIterator struct {
	rows []kv
	idx  int
}

func (it *fakeIterator) Next() bool {
	if it.idx >= len(it.rows) {
		return false
	}
	it.idx++
	return true
}

func (it *fakeIterator) Key() []byte {
	if it.idx == 0 || it.idx > len(it.rows) {
		return nil
	}
	return it.rows[it.idx-1].key
}

func (it *fakeIterator) Value() []byte {
	if it.idx == 0 || it.idx > len(it.rows) {
		return nil
	}
	return it.rows[it.idx-1].value
}

func (it *fakeIterator) Err() error {
	return nil
}

func (it *fakeIterator) Close() error {
	return nil
}
