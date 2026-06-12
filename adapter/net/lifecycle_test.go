//go:build !chaos_off

package net_test

import (
	"net"
	"testing"

	chaosnet "github.com/RomanAgaltsev/chaotic/adapter/net"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each Read, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by Read running After.
func TestReadReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	a, b := net.Pipe()
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	c := chaosnet.WrapConn(a, eng)

	for range 3 {
		go func() { _, _ = b.Write([]byte("x")) }()
		if _, err := c.Read(make([]byte, 1)); err != nil {
			t.Fatalf("Read err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential reads; the max-concurrent slot is leaking", got)
	}
}
