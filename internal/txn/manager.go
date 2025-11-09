package txn

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Manager coordinates transaction lifecycle and hooks log/recovery.
type Manager struct {
	mu   sync.Mutex
	seq  uint64
	txns map[uint64]State
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

func NewManager() *Manager {
	return &Manager{txns: make(map[uint64]State)}
}

func (m *Manager) Begin(ctx context.Context) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	state := State{ID: m.seq, StartedAt: time.Now(), Status: StatusActive}
	m.txns[state.ID] = state
	return state, nil
}

func (m *Manager) Commit(id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.txns[id]
	if !ok {
		return ErrUnknownTxn
	}

	state.Status = StatusCommitted
	m.txns[id] = state
	return nil
}

func (m *Manager) Rollback(id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.txns[id]
	if !ok {
		return ErrUnknownTxn
	}

	state.Status = StatusAborted
	m.txns[id] = state
	return nil
}

var ErrUnknownTxn = errors.New("transaction not found")
