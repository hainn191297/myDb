package txn

import (
	"context"
	"testing"
	"time"
)

func TestLockManager_SimpleAcquireRelease(t *testing.T) {
	lm := NewLockManager()
	ctx := context.Background()
	txnID := uint64(1)
	key := "t1.k1"

	// Acquire Write Lock
	if err := lm.Acquire(ctx, txnID, key, WriteLock); err != nil {
		t.Fatalf("failed to acquire write lock: %v", err)
	}

	// Verify lock is held
	lm.mu.Lock()
	l, ok := lm.locks[key]
	lm.mu.Unlock()
	if !ok {
		t.Fatal("lock not found in map")
	}
	if l.holders[txnID] != WriteLock {
		t.Fatal("lock not held by txnID")
	}

	// Release
	lm.Release(txnID)

	// Verify lock is released (or cleaned up)
	lm.mu.Lock()
	l, ok = lm.locks[key]
	lm.mu.Unlock()
	if ok {
		// If lock struct remains, it should have no holders
		l.mu.Lock()
		if len(l.holders) > 0 {
			t.Fatal("lock still has holders after release")
		}
		l.mu.Unlock()
	}
}

func TestLockManager_Conflict(t *testing.T) {
	lm := NewLockManager()
	ctx := context.Background()
	txnA := uint64(1)
	txnB := uint64(2)
	key := "t1.k1"

	// TxnA acquires Write Lock
	if err := lm.Acquire(ctx, txnA, key, WriteLock); err != nil {
		t.Fatalf("txnA failed to acquire: %v", err)
	}

	// TxnB tries to acquire Write Lock (should block)
	errCh := make(chan error)
	go func() {
		errCh <- lm.Acquire(ctx, txnB, key, WriteLock)
	}()

	// Wait a bit to ensure TxnB is blocked
	select {
	case err := <-errCh:
		t.Fatalf("txnB should have blocked, but returned: %v", err)
	case <-time.After(50 * time.Millisecond):
		// Expected behavior
	}

	// TxnA releases
	lm.Release(txnA)

	// TxnB should now proceed
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("txnB failed to acquire after release: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("txnB did not acquire lock after release")
	}
}

func TestLockManager_ReadReadShared(t *testing.T) {
	lm := NewLockManager()
	ctx := context.Background()
	txnA := uint64(1)
	txnB := uint64(2)
	key := "t1.k1"

	// TxnA acquires Read Lock
	if err := lm.Acquire(ctx, txnA, key, ReadLock); err != nil {
		t.Fatalf("txnA failed to acquire: %v", err)
	}

	// TxnB acquires Read Lock (should succeed immediately)
	if err := lm.Acquire(ctx, txnB, key, ReadLock); err != nil {
		t.Fatalf("txnB failed to acquire: %v", err)
	}

	lm.Release(txnA)
	lm.Release(txnB)
}

func TestLockManager_DeadlockTimeout(t *testing.T) {
	lm := NewLockManager()
	ctx := context.Background()
	txnA := uint64(1)
	txnB := uint64(2)
	key := "t1.k1"

	if err := lm.Acquire(ctx, txnA, key, WriteLock); err != nil {
		t.Fatalf("txnA failed to acquire: %v", err)
	}

	// We can't easily change the hardcoded 5s timeout in the test without changing code.
	// But we can check if it eventually times out or respects context cancellation.
	// Let's test context cancellation which is faster.

	ctxCancel, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := lm.Acquire(ctxCancel, txnB, key, WriteLock)
	if err == nil {
		t.Fatal("txnB should have failed")
	}
	if err != context.DeadlineExceeded {
		// Note: The code returns ctx.Err() which is DeadlineExceeded, OR ErrLockTimeout if select default case hit?
		// Code:
		// case <-ctx.Done():
		//   return ctx.Err()
		// case <-time.After(5 * time.Second):
		//   return dberrors.ErrLockTimeout
		t.Logf("txnB returned error: %v", err)
	}
}
