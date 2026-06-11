package engine

import (
	"context"
	"testing"
	"time"
)

func TestMatchTimeWindowInsideAndOutside(t *testing.T) {
	// A fixed "now" at 03:00 UTC; window 02:00–04:00 contains it.
	at := func(h, m int) time.Time {
		return time.Date(2026, 6, 7, h, m, 0, 0, time.UTC)
	}
	r := NewRule(matchTimeWindowAt(2, 0, 4, 0, func() time.Time { return at(3, 0) }))
	if !r.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("03:00 should be inside the 02:00–04:00 window")
	}

	r2 := NewRule(matchTimeWindowAt(2, 0, 4, 0, func() time.Time { return at(5, 0) }))
	if r2.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("05:00 should be outside the 02:00–04:00 window")
	}
}

func TestMatchTimeWindowPublicConstructorUsesNow(t *testing.T) {
	// The public MatchTimeWindow compiles and produces a usable matcher; we
	// can't assert wall-clock timing, only that a degenerate full-day window
	// always matches.
	r := NewRule(MatchTimeWindow(0, 0, 23, 59))
	if !r.matches(context.Background(), Op{Kind: OpHTTPClient}) {
		t.Fatal("a 00:00–23:59 window should match at any time of day")
	}
}
