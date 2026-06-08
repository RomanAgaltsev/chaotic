//go:build !chaos_off

package kafka

import (
	"context"
	"testing"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each write, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by WriteMessages running After.
func TestWriteMessagesReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpKafka),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	w := newWriter(&fakeWriter{}, eng)
	ctx := context.Background()
	for range 3 {
		if err := w.WriteMessages(ctx, kafkago.Message{Value: []byte("a")}); err != nil {
			t.Fatalf("WriteMessages err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential writes; the max-concurrent slot is leaking", got)
	}
}
