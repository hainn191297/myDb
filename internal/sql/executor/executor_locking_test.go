package executor

import (
	"context"
	"testing"
	"time"

	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/txn"
)

// TestExecutorLocking_UpdateConflict verifies that two transactions
// attempting to update the same row are serialized by locking.
func TestExecutorLocking_UpdateConflict(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Environment
	mgr := txn.NewManager(nil, nil)
	provider := newFakeProvider()

	// Create Catalog engines
	provider.engines[schema.SystemSchema+"."+schema.CatalogTablesTable] = newFakeEngine()
	provider.engines[schema.SystemSchema+"."+schema.CatalogIndexesTable] = newFakeEngine()

	// Initialize Catalog
	cat := schema.NewCatalog(
		provider.engines[schema.SystemSchema+"."+schema.CatalogTablesTable],
		provider.engines[schema.SystemSchema+"."+schema.CatalogIndexesTable],
	)

	// Create User Table Engine explicitly for fakeProvider
	provider.engines["public.users"] = newFakeEngine()

	// Create User Table
	// "users" (id INT PRIMARY KEY, name TEXT)
	// We cheat by injecting directly into catalog and engine to skip CreateTable overhead if possible,
	// but using CreateTableOp is cleaner.

	createTablePlan := planner.Plan{
		Root: &planner.CreateTableOp{
			Schema: "public",
			Table:  "users",
			Columns: []schema.ColumnDef{
				{Name: "id", Type: schema.TypeInt64, PrimaryKey: true},
				{Name: "name", Type: schema.TypeText},
			},
		},
	}
	execInit := New(createTablePlan, Options{Catalog: cat, Provider: provider})
	if _, err := execInit.Next(ctx); err != nil {
		t.Fatalf("setup create table: %v", err)
	}

	// Insert Data: (1, 'alice')

	val1, _ := schema.TypeInt64.Encode("1")
	valAlice, _ := schema.TypeText.Encode("alice")
	valBob, _ := schema.TypeText.Encode("bob")

	insertPlan := planner.Plan{
		Root: &planner.InsertOp{
			Schema: "public",
			Table:  "users",
			Columns: []string{"id", "name"},
			Values: [][][]byte{{val1, valAlice}},
		},
	}
	execInit = New(insertPlan, Options{Catalog: cat, Provider: provider})
	if _, err := execInit.Next(ctx); err != nil {
		t.Fatalf("setup insert: %v", err)
	}

	// 2. Prepare Transactions

	// Update Plan: UPDATE users SET name='bob' WHERE id=1
	// We need manual SetClauses and Filter.
	// Filter "id = 1"
	// SetClauses: "name" -> encoded "bob"
	updatePlan := planner.Plan{
		Root: &planner.UpdateOp{
			Schema: "public",
			Table:  "users",
			SetClauses: map[string][]byte{
				"name": valBob,
			},
			Filter: "id = 1", // The executor matchesFilter handles this string
		},
	}

	// T1 Begin
	sess1 := &SessionTxn{}
	exec1 := New(planner.Plan{Root: &planner.TxnOp{Action: planner.TxnBegin}}, Options{TxnManager: mgr, SessionTxn: sess1})
	exec1.Next(ctx)
	t1ID := sess1.Current.ID

	// T2 Begin
	sess2 := &SessionTxn{}
	exec2 := New(planner.Plan{Root: &planner.TxnOp{Action: planner.TxnBegin}}, Options{TxnManager: mgr, SessionTxn: sess2})
	exec2.Next(ctx)
	t2ID := sess2.Current.ID

	t.Logf("Started T1=%d, T2=%d", t1ID, t2ID)

	// 3. Execution

	// T1 executes Update (should acquire WriteLock on key for id=1)
	exec1 = New(updatePlan, Options{TxnManager: mgr, SessionTxn: sess1, Provider: provider, Catalog: cat})
	if _, err := exec1.Next(ctx); err != nil {
		t.Fatalf("T1 update failed: %v", err)
	}
	t.Log("T1 updated row")

	// T2 executes Update (should BLOCK)
	doneCh := make(chan error)
	go func() {
		exec2 = New(updatePlan, Options{TxnManager: mgr, SessionTxn: sess2, Provider: provider, Catalog: cat})
		_, err := exec2.Next(ctx)
		doneCh <- err
	}()

	// Verify T2 is blocked
	select {
	case err := <-doneCh:
		t.Errorf("T2 should have blocked, but finished with err=%v", err)
	case <-time.After(100 * time.Millisecond):
		t.Log("T2 is blocked as expected")
	}

	// 4. Commit T1
	exec1 = New(planner.Plan{Root: &planner.TxnOp{Action: planner.TxnCommit}}, Options{TxnManager: mgr, SessionTxn: sess1})
	if _, err := exec1.Next(ctx); err != nil {
		t.Fatalf("T1 commit failed: %v", err)
	}
	t.Log("T1 committed")

	// 5. Verify T2 unblocks and proceeds
	select {
	case err := <-doneCh:
		if err != nil {
			t.Fatalf("T2 failed after unblocking: %v", err)
		}
		t.Log("T2 finished successfully")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("T2 timed out waiting for lock")
	}
}
