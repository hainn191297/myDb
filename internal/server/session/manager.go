package session

import (
    "context"
    "sync"
    "time"
)

// Manager tracks client sessions and enforces limits defined in config.
type Manager struct {
    maxSessions int
    expiry      time.Duration

    mu       sync.Mutex
    sessions map[string]Session
}

// Session is a lightweight record for now; it will expand with txn state later.
type Session struct {
    ID        string
    CreatedAt time.Time
    LastSeen  time.Time
}

func NewManager(max int, expiry time.Duration) *Manager {
    return &Manager{
        maxSessions: max,
        expiry:      expiry,
        sessions:    make(map[string]Session),
    }
}

// Create registers a new session if capacity allows.
func (m *Manager) Create(ctx context.Context) (Session, error) {
    select {
    case <-ctx.Done():
        return Session{}, ctx.Err()
    default:
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    if len(m.sessions) >= m.maxSessions {
        return Session{}, ErrSessionLimit
    }

    now := time.Now()
    sess := Session{
        ID:        generateID(),
        CreatedAt: now,
        LastSeen:  now,
    }
    m.sessions[sess.ID] = sess
    return sess, nil
}

// Touch updates LastSeen to keep-alive the session.
func (m *Manager) Touch(id string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    sess, ok := m.sessions[id]
    if !ok {
        return
    }
    sess.LastSeen = time.Now()
    m.sessions[id] = sess
}

// Sweep removes expired sessions; intended for a background ticker.
func (m *Manager) Sweep() {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now()
    for id, sess := range m.sessions {
        if now.Sub(sess.LastSeen) > m.expiry {
            delete(m.sessions, id)
        }
    }
}

var ErrSessionLimit = fmtError("session limit reached")

// fmtError is a placeholder until an error package is added.
type fmtError string

func (e fmtError) Error() string { return string(e) }

// generateID will become cryptographically strong later.
func generateID() string {
    return time.Now().Format("20060102T150405.000000000")
}
