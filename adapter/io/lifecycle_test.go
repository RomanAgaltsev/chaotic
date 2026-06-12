//go:build !chaos_off

package io_test

import (
	"strings"
	"testing"

	chaosio "github.com/RomanAgaltsev/chaotic/adapter/io"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each Read, or chaos silently
// stops once the cap is reached. A truncate sentinel takes the shaped branch;
// assert After still runs there.
func TestReadReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Always(),
		engine.WithFault(fault.SlowReader(1_000_000)), // negligible delay, shaped branch
	).Named("slow"))

	for range 3 {
		r := chaosio.WrapReader(strings.NewReader("x"), eng)
		if _, err := r.Read(make([]byte, 1)); err != nil {
			t.Fatalf("Read err = %v", err)
		}
	}
	if got := eng.Hits("slow"); got != 3 {
		t.Fatalf("rule fired %d/3 reads; the max-concurrent slot is leaking", got)
	}
}
