//go:build !chaos_off

package main

import "testing"

func TestBreakerStopsCallingAfterThreshold(t *testing.T) {
	calls := drive(10)
	if calls != 3 {
		t.Fatalf("dependency was called %d times, want 3 (breaker should open after the threshold)", calls)
	}
}
