package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestNowWithoutSkew(t *testing.T) {
	got := engine.Now(context.Background())
	if d := time.Since(got); d < -time.Second || d > time.Second {
		t.Fatalf("Now without skew = %v, want ≈ time.Now()", got)
	}
}

func TestNowWithSkew(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	_ = fault.Clock(2 * time.Hour).Apply(ctx)
	want := time.Now().Add(2 * time.Hour)
	got := engine.Now(ctx)
	if d := got.Sub(want); d < -time.Second || d > time.Second {
		t.Fatalf("Now = %v, want ≈ %v (diff %v)", got, want, d)
	}
}

func TestSinceAndUntil(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	_ = fault.Clock(time.Hour).Apply(ctx)
	base := time.Now()
	// Now is ~1h ahead of base, so Since(base) ≈ +1h and Until(base) ≈ -1h.
	if s := engine.Since(ctx, base); s < 59*time.Minute || s > 61*time.Minute {
		t.Fatalf("Since = %v, want ≈ 1h", s)
	}
	if u := engine.Until(ctx, base); u > -59*time.Minute || u < -61*time.Minute {
		t.Fatalf("Until = %v, want ≈ -1h", u)
	}
}
