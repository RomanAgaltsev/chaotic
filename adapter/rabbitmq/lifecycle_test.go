//go:build !chaos_off

package rabbitmq

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
	amqp "github.com/rabbitmq/amqp091-go"
)

// A WithMaxConcurrent slot must be released after each publish, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by PublishWithContext running After.
func TestPublishReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRabbitMQ),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	c := newChannel(&fakeChannel{}, eng)
	ctx := context.Background()
	for range 3 {
		if err := c.PublishWithContext(ctx, "ex", "rk", false, false, amqp.Publishing{}); err != nil {
			t.Fatalf("publish err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential publishes; the max-concurrent slot is leaking", got)
	}
}
