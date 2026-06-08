//go:build chaos_off

package kafka

import (
	"context"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	w := &Writer{w: &fakeWriter{}}
	ctx := context.Background()
	msgs := []kafkago.Message{{Value: []byte("a")}}
	avg := testing.AllocsPerRun(100, func() {
		_ = w.WriteMessages(ctx, msgs...)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
