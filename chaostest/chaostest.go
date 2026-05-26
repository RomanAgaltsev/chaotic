// Package chaostest provides testing.TB integration helpers for chaotic.
// The most-used function is New, which returns a fresh engine bound to the
// test's t.Cleanup so faults from one test never leak into another.
package chaostest

import (
	"sort"
	"testing"

	"github.com/ag4r/chaotic/engine"
)

// New returns an engine that resets itself when t finishes.
// Safe for t.Parallel because every test gets its own engine.
func New(t testing.TB, opts ...engine.Option) *engine.Engine {
	t.Helper()
	e := engine.New(opts...)
	t.Cleanup(e.Reset)
	return e
}

// AssertHits fails t with a clear message if eng has not fired ruleName
// exactly want times.
func AssertHits(t testing.TB, eng *engine.Engine, ruleName string, want int) {
	t.Helper()
	if got := eng.Hits(ruleName); got != want {
		t.Errorf("chaotic: rule %q fired %d times, want %d", ruleName, got, want)
	}
}

// AssertEventsExhausted fails t if any named rule on eng has not fired at
// least once. Useful for asserting "every chaos rule I configured was
// actually exercised by my test".
func AssertEventsExhausted(t testing.TB, eng *engine.Engine) {
	t.Helper()
	hits := eng.AllHits()
	var unfired []string
	for name, n := range hits {
		if n == 0 {
			unfired = append(unfired, name)
		}
	}
	if len(unfired) == 0 {
		return
	}
	sort.Strings(unfired)
	t.Errorf("chaotic: %d rule(s) never fired: %v", len(unfired), unfired)
}
