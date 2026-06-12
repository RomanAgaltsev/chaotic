//go:build !chaos_off

package chaos_test

import (
	"context"
	"testing"
	"time"

	"github.com/RomanAgaltsev/chaotic/chaos"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func newClockEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.WithFault(fault.Clock(3*time.Hour)),
		engine.MatchName("deadline.compute"),
	))
}

func TestPointFiresClockSkewsNow(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), newClockEngine())

	before := engine.Now(ctx)
	if d := time.Since(before); d < -time.Second || d > time.Second {
		t.Fatalf("before firing, Now = %v, want ≈ real time", before)
	}

	if err := chaos.Point(ctx, "deadline.compute"); err != nil {
		t.Fatalf("Point returned %v, want nil", err)
	}

	want := time.Now().Add(3 * time.Hour)
	got := engine.Now(ctx)
	if d := got.Sub(want); d < -time.Second || d > time.Second {
		t.Fatalf("after firing, Now = %v, want ≈ %v", got, want)
	}

	// Sticky: a later read is still skewed (not before the first skewed read).
	if got2 := engine.Now(ctx); got2.Before(got) {
		t.Fatalf("second read %v before first %v; skew not sticky", got2, got)
	}
}

func TestUnmatchedPointLeavesClockUnskewed(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), newClockEngine())
	_ = chaos.Point(ctx, "other.point") // does not match the rule
	if got := fault.Skew(ctx); got != 0 {
		t.Fatalf("Skew = %v, want 0 (no matching rule fired)", got)
	}
}

func TestResetClockClearsSkew(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), newClockEngine())
	_ = chaos.Point(ctx, "deadline.compute")
	fault.ResetClock(ctx)
	if got := fault.Skew(ctx); got != 0 {
		t.Fatalf("Skew after reset = %v, want 0", got)
	}
}
