package curl

import (
	"errors"
	"sync"
)

var ErrHandlePoolClosed = errors.New("curl: handle pool is closed")

// HandleLease represents exclusive ownership of one native easy handle.
type HandleLease struct {
	pool *HandlePool
	id   int
	once sync.Once
}

func (l *HandleLease) ID() int {
	if l == nil {
		return 0
	}
	return l.id
}

func (l *HandleLease) Release() {
	if l == nil || l.pool == nil {
		return
	}
	l.once.Do(func() {
		l.pool.release(l.id)
	})
}

// HandlePool models per-request exclusive easy-handle ownership.
type HandlePool struct {
	mu     sync.Mutex
	nextID int
	free   []int
	closed bool
}

func NewHandlePool() *HandlePool {
	return &HandlePool{}
}

func (p *HandlePool) Lease() (*HandleLease, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, ErrHandlePoolClosed
	}
	var id int
	if len(p.free) > 0 {
		id = p.free[len(p.free)-1]
		p.free = p.free[:len(p.free)-1]
	} else {
		p.nextID++
		id = p.nextID
	}
	return &HandleLease{pool: p, id: id}, nil
}

func (p *HandlePool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.free = nil
}

func (p *HandlePool) release(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || id == 0 {
		return
	}
	p.free = append(p.free, id)
}
