package engine

import (
	"sync"
	"time"
)

// StickyAttr makes a rule "sticky" by an Op attribute: once the rule fires for a
// given Attrs[key] value, every later matching Op carrying that same value keeps
// firing for window, bypassing the counter. This models a resource that is stuck
// in a degraded state (e.g. one user wedged after a fault). cap bounds memory:
// at most cap distinct values are tracked, and the least recently marked value
// is evicted to make room. A key absent from an Op's Attrs is never sticky.
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

// stickyTracker is a bounded value->expiry set keyed by an Op attribute. Expiry
// doubles as the eviction order: window is constant, so the entry expiring
// soonest is exactly the one marked longest ago.
type stickyTracker struct {
	key    string
	window time.Duration
	cap    int
	now    func() time.Time

	mu   sync.Mutex
	seen map[string]time.Time // attr value -> expiry
}

// sticky reports whether op's attr value is currently within its sticky window.
// An expired value is dropped on the way past.
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

// mark records (or refreshes) op's attr value as sticky for the window. A
// refresh rewrites the expiry, which is also the eviction key, so a hot value
// cannot be evicted while a colder one survives. When a new value arrives at
// capacity, the least recently marked entry is evicted.
func (s *stickyTracker) mark(op Op) {
	v, ok := op.Attrs[s.key]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.seen[v]; !exists && len(s.seen) >= s.cap {
		s.evictOldestLocked()
	}
	s.seen[v] = s.now().Add(s.window)
}

// evictOldestLocked removes the entry whose sticky window ends soonest. The
// caller must hold s.mu and must have established that s.seen is non-empty. The
// scan runs only on an insert at capacity, and cap is small by construction —
// it exists to bound memory. Entries sharing the soonest expiry were marked at
// the same instant, so any of them is an equally correct victim.
//
// No //bigo:max budget here: the scan is O(len(s.seen)), but bigo cannot price
// a range over a map, so an annotation would fail the complexity gate rather
// than document anything.
func (s *stickyTracker) evictOldestLocked() {
	var (
		oldest    string
		oldestExp time.Time
		first     = true
	)
	for v, exp := range s.seen {
		if first || exp.Before(oldestExp) {
			oldest, oldestExp, first = v, exp, false
		}
	}
	delete(s.seen, oldest)
}
