package engine

import (
	"sync"
	"time"
)

// tokenBucket is a minimal in-tree rate limiter. keeping the core module
// dependency-free. It refills at rps tokens/seconds with burst capacity rps.
type tokenBucket struct {
	mu     sync.Mutex
	tokens float64
	max    float64
	rps    float64
	last   time.Time
}

func newTokenBucket(rps int) *tokenBucket {
	if rps < 1 {
		panic("chaotic: WithRateLimit rps must be >= 1")
	}
	r := float64(rps)
	return &tokenBucket{
		tokens: r,
		max:    r,
		rps:    r,
		last:   time.Now(),
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.rps
	if b.tokens > b.max {
		b.tokens = b.max
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
