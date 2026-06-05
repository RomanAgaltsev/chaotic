//go:build chaos_off

package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestZeroAllocPublishUnderChaosOff(t *testing.T) {
	c := &Channel{ch: &fakeChannel{}}
	ctx := context.Background()
	msg := amqp.Publishing{}
	avg := testing.AllocsPerRun(100, func() {
		_ = c.PublishWithContext(ctx, "ex", "rk", false, false, msg)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
