package engine

import "sync"

// failureBudget tracks the error rate over a fixed-size sliding window of the
// last len(ring) call outcomes. overBudget reports true only once the window
// is full, so early calls are never suppressed for lack of data.
type failureBudget struct {
	maxRate float64

	mu     sync.Mutex
	ring   []bool // true == error
	idx    int
	filled bool
	errors int
}

func newFailureBudget(maxRate float64, window int) *failureBudget {
	if maxRate < 0 || maxRate > 1 {
		panic("chaotic: WithFailureBudget maxErrorRate must be in [0,1]")
	}
	if window < 1 {
		panic("chaotic: WithFailureBudget window must be >= 1")
	}
	return &failureBudget{maxRate: maxRate, ring: make([]bool, window)}
}

func (b *failureBudget) record(callErr error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ring[b.idx] {
		b.errors--
	}
	isErr := callErr != nil
	b.ring[b.idx] = isErr
	if isErr {
		b.errors++
	}
	b.idx++
	if b.idx == len(b.ring) {
		b.idx = 0
		b.filled = true
	}
}

func (b *failureBudget) overBudget() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.filled {
		return false
	}
	return float64(b.errors)/float64(len(b.ring)) >= b.maxRate
}
