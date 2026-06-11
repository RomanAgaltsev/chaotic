package engine

import (
	"sync"
	"time"
)

// StickyAttr makes a rule "sticky" by an Op attribute: once the rule fires for a
// given Attrs[key] value, every later matching Op carrying that same value keeps
// firing for window, bypassing the counter. This models a resource that is stuck
// in a degraded state (e.g. one user wedged after a fault). cap bounds memory:
// at most cap distinct values are tracked, evicted FIFO. A key absent from an
// Op's Attrs is never sticky.
func StickyAttr(key string, window time.Duration, cap int) RuleOption {
	if cap < 1 {
		cap = 1
	}
	return func(r *Rule) {
		r.sticky = &stickyTracker{
			key:    key,
			window: window,
			cap:    cap,
			seen:   make(map[string]time.Time, cap),
			now:    time.Now,
		}
	}
}

// stickyTracker is a bounded value->expiry set keyed by an Op attribute.
type stickyTracker struct {
	key    string
	window time.Duration
	cap    int
	now    func() time.Time

	mu    sync.Mutex
	seen  map[string]time.Time // attr value -> expiry
	order []string             // FIFO of values for eviction
}

// sticky reports whether op's attr value is currently within its sticky window.
func (s *stickyTracker) sticky(op Op) bool {
	v, ok := op.Attrs[s.key]
	if !ok {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.seen[v]
	if !ok {
		return false
	}
	if s.now().After(exp) {
		delete(s.seen, v)
		return false
	}
	return true
}

// mark records (or refreshes) op's attr value as sticky for the window.
func (s *stickyTracker) mark(op Op) {
	v, ok := op.Attrs[s.key]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.seen[v]; !exists {
		if len(s.order) >= s.cap {
			oldest := s.order[0]
			s.order = s.order[1:]
			delete(s.seen, oldest)
		}
		s.order = append(s.order, v)
	}
	s.seen[v] = s.now().Add(s.window)
}
