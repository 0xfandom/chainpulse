package mcp

import (
	"sync"
)

// session is one active SSE subscriber: the channel the transport pushes
// outbound frames into and a done flag set when the subscriber exits.
type session struct {
	id      string
	out     chan []byte
	closed  chan struct{}
	closeMu sync.Mutex
}

// closeOnce flips done exactly once; safe to call from both the producer
// (POST handler) and the consumer (SSE goroutine).
func (s *session) closeOnce() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
	}
}

// sessionMap is the in-memory registry of active SSE subscribers.
type sessionMap struct {
	mu sync.RWMutex
	m  map[string]*session
}

func newSessionMap() *sessionMap {
	return &sessionMap{m: make(map[string]*session)}
}

func (sm *sessionMap) add(s *session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[s.id] = s
}

func (sm *sessionMap) get(id string) (*session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.m[id]
	return s, ok
}

func (sm *sessionMap) remove(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.m, id)
}

func (sm *sessionMap) len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.m)
}
