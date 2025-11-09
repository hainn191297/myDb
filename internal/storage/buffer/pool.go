package buffer

import "sync"

// Pool is a stub for the buffer pool / page cache.
type Pool struct {
    mu   sync.Mutex
    data map[string][]byte
}

func NewPool() *Pool {
    return &Pool{data: make(map[string][]byte)}
}

func (p *Pool) Get(key string) ([]byte, bool) {
    p.mu.Lock()
    defer p.mu.Unlock()
    v, ok := p.data[key]
    return v, ok
}

func (p *Pool) Set(key string, val []byte) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.data[key] = val
}

func (p *Pool) Evict(key string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.data, key)
}
