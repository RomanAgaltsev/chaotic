//go:build !chaos_off

package nats

import (
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each publish, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by Publish running After.
func TestPublishReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	c := newConn(&fakeConn{}, eng)
	for range 3 {
		if err := c.Publish("events", []byte("hi")); err != nil {
			t.Fatalf("Publish err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential publishes; the max-concurrent slot is leaking", got)
	}
}
