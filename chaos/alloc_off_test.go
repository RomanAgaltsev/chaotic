//go:build chaos_off

package chaos_test

import (
	"context"
	"testing"

	"github.com/ag4r/chaotic/chaos"
	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), engine.New())
	avg := testing.AllocsPerRun(100, func() {
		_ = chaos.Point(ctx, "p")
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
