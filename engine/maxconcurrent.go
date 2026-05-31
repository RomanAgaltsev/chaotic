package engine

// semaphore is a counting semaphore bounding concurrent faulted calls.
type semaphore struct {
	ch chan struct{}
}

func newSemaphore(n int) *semaphore {
	if n < 1 {
		panic("chaotic: WithMaxConcurrent n must be >= 1")
	}
	return &semaphore{make(chan struct{}, n)}
}

// tryAcquire returns a release func and true if a slot was free, else nil and
// false. The release func is idempotent only via the caller's sync.Once.
func (s *semaphore) tryAcquire() (func(), bool) {
	select {
	case s.ch <- struct{}{}:
		return func() { <-s.ch }, true
	default:
		return nil, false
	}
}
