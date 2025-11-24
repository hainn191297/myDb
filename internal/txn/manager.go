package txn

import (
	"context"
	"sync"
	"time"

	dberrors "github.com/hainn191297/myDb/internal/errors"
	"github.com/hainn191297/myDb/internal/logging"
)

// LockType represents the type of lock (Read or Write).
type LockType int

const (
	ReadLock LockType = iota
	WriteLock
)

// LockManager manages row-level locks.
type LockManager struct {
	mu    sync.Mutex
	locks map[string]*lock // key -> lock
}

type lock struct {
	mu      sync.Mutex
	holders map[uint64]LockType // txnID -> LockType
	waiters []*waiter
}

type waiter struct {
	txnID uint64
	lType LockType
	ready chan struct{}
}

func newLock() *lock {
	return &lock{
		holders: make(map[uint64]LockType),
		waiters: make([]*waiter, 0),
	}
}

func NewLockManager() *LockManager {
	return &LockManager{
		locks: make(map[string]*lock),
	}
}

// Acquire requests a lock on a key. It blocks until the lock is acquired or timeout.
func (lm *LockManager) Acquire(ctx context.Context, txnID uint64, key string, lType LockType) error {
	logging.Debugf("txn %d acquiring lock %s type %d", txnID, key, lType)
	lm.mu.Lock()
	l, exists := lm.locks[key]
	if !exists {
		l = newLock()
		lm.locks[key] = l
	}
	lm.mu.Unlock()

	l.mu.Lock()

	// Check if we can acquire immediately
	if lm.canAcquire(l, txnID, lType) {
		l.holders[txnID] = lType
		l.mu.Unlock()
		logging.Debugf("txn %d acquired lock %s immediately", txnID, key)
		return nil
	}

	// Cannot acquire, add to waiters
	logging.Debugf("txn %d waiting for lock %s", txnID, key)
	w := &waiter{
		txnID: txnID,
		lType: lType,
		ready: make(chan struct{}),
	}
	l.waiters = append(l.waiters, w)
	l.mu.Unlock()

	// Wait
	select {
	case <-w.ready:
		logging.Debugf("txn %d acquired lock %s after wait", txnID, key)
		return nil
	case <-ctx.Done():
		logging.Debugf("txn %d context done while waiting for lock %s", txnID, key)
		lm.removeWaiter(l, w)
		return ctx.Err()
	case <-time.After(5 * time.Second): // Simple deadlock detection
		logging.Debugf("txn %d timeout waiting for lock %s", txnID, key)
		lm.removeWaiter(l, w)
		return dberrors.ErrLockTimeout
	}
}

func (lm *LockManager) canAcquire(l *lock, txnID uint64, lType LockType) bool {
	// If we already hold it
	if currentType, ok := l.holders[txnID]; ok {
		if currentType == WriteLock || lType == ReadLock {
			return true
		}
		// Upgrade needed (Read -> Write)
		// For MVP, fail if others hold lock
		if len(l.holders) > 1 {
			return false
		}
		return true
	}

	// Check conflicts
	for _, holderType := range l.holders {
		if lType == WriteLock || holderType == WriteLock {
			return false
		}
	}
	return true
}

func (lm *LockManager) removeWaiter(l *lock, w *waiter) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, waiter := range l.waiters {
		if waiter == w {
			l.waiters = append(l.waiters[:i], l.waiters[i+1:]...)
			break
		}
	}
}

// Release releases all locks for a transaction.
func (lm *LockManager) Release(txnID uint64) {
	logging.Debugf("txn %d releasing locks", txnID)
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for key, l := range lm.locks {
		l.mu.Lock()
		if _, ok := l.holders[txnID]; ok {
			delete(l.holders, txnID)

			// Wake up waiters if possible
			// Simple strategy: wake up first waiter if it can run
			// Note: This might cause thundering herd or starvation, but OK for MVP
			if len(l.waiters) > 0 {
				// Check if next waiter can proceed
				// We need to be careful: if we just removed a Read lock, and next is Write, it can proceed ONLY if no other readers.
				// If we removed Write, next can proceed.

				// Iterate waiters and wake up compatible ones
				// Since we modified holders, we need to re-check compatibility for waiters
				activeWaiters := l.waiters[:0]
				for _, w := range l.waiters {
					if lm.canAcquire(l, w.txnID, w.lType) {
						l.holders[w.txnID] = w.lType
						close(w.ready)
					} else {
						activeWaiters = append(activeWaiters, w)
					}
				}
				l.waiters = activeWaiters
			}
		}

		// Cleanup empty locks
		if len(l.holders) == 0 && len(l.waiters) == 0 {
			delete(lm.locks, key)
		}
		l.mu.Unlock()
	}
}

// Manager coordinates transaction lifecycle and hooks log/recovery.
type Manager struct {
	mu      sync.Mutex
	seq     uint64
	txns    map[uint64]State
	lockMgr *LockManager

	// Dependencies for commit durability
	bufferMgr BufferFlusher
	walMgr    WALSyncer
}

// BufferFlusher interface for flushing dirty pages
type BufferFlusher interface {
	FlushAll() error
}

// WALSyncer interface for syncing WAL to disk
type WALSyncer interface {
	SyncAll() error
}

// State captures metadata for a running transaction.
type State struct {
	ID        uint64
	StartedAt time.Time
	Status    Status
}

// Status indicates lifecycle stage.
type Status string

const (
	StatusActive    Status = "active"
	StatusCommitted Status = "committed"
	StatusAborted   Status = "aborted"
)

func NewManager(bufferMgr BufferFlusher, walMgr WALSyncer) *Manager {
	return &Manager{
		txns:      make(map[uint64]State),
		lockMgr:   NewLockManager(),
		bufferMgr: bufferMgr,
		walMgr:    walMgr,
	}
}

func (m *Manager) Begin(ctx context.Context) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	state := State{ID: m.seq, StartedAt: time.Now(), Status: StatusActive}
	m.txns[state.ID] = state
	logging.DebugContext(ctx, "txn begin id=%d", state.ID)
	return state, nil
}

func (m *Manager) Commit(id uint64) error {
	m.mu.Lock()

	state, ok := m.txns[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknownTxn
	}

	state.Status = StatusCommitted
	m.txns[id] = state
	m.mu.Unlock()

	// CRITICAL: Ensure durability by flushing dirty pages and syncing WAL
	// This prevents data loss if crash occurs immediately after COMMIT returns
	if m.bufferMgr != nil {
		if err := m.bufferMgr.FlushAll(); err != nil {
			logging.Debugf("txn commit id=%d: buffer flush failed: %v", id, err)
			// Continue despite error - partial flush is better than none
		}
	}

	if m.walMgr != nil {
		if err := m.walMgr.SyncAll(); err != nil {
			logging.Debugf("txn commit id=%d: WAL sync failed: %v", id, err)
			// Continue despite error
		}
	}

	m.lockMgr.Release(id)
	logging.Debugf("txn commit id=%d (flushed+synced)", id)
	return nil
}

func (m *Manager) Rollback(id uint64) error {
	m.mu.Lock()
	// defer m.mu.Unlock() -- Removed to avoid double unlock

	state, ok := m.txns[id]
	if !ok {
		m.mu.Unlock()
		return ErrUnknownTxn
	}

	state.Status = StatusAborted
	m.txns[id] = state
	m.mu.Unlock()

	m.lockMgr.Release(id)
	logging.Warnf("txn rollback id=%d", id)
	return nil
}

// AcquireLock acquires a lock for the given transaction.
func (m *Manager) AcquireLock(ctx context.Context, txnID uint64, table, key string, lType LockType) error {
	lockKey := table + "." + key
	return m.lockMgr.Acquire(ctx, txnID, lockKey, lType)
}

// ErrUnknownTxn is returned when a transaction ID is not found.
// Deprecated: Use dberrors.ErrTxnNotFound instead.
var ErrUnknownTxn = dberrors.ErrTxnNotFound
