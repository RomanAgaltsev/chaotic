package fault_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ag4r/chaotic/fault"
)

func TestWithClockStartsAtZero(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	if got := fault.Skew(ctx); got != 0 {
		t.Fatalf("Skew = %v, want 0", got)
	}
}

func TestClockSetsSkew(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	if err := fault.Clock(2 * time.Hour).Apply(ctx); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
	if got := fault.Skew(ctx); got != 2*time.Hour {
		t.Fatalf("Skew = %v, want 2h", got)
	}
}

func TestResetClock(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	_ = fault.Clock(time.Hour).Apply(ctx)
	fault.ResetClock(ctx)
	if got := fault.Skew(ctx); got != 0 {
		t.Fatalf("Skew after reset = %v, want 0", got)
	}
}

func TestClockNoCellIsNoop(t *testing.T) {
	ctx := context.Background() // no cell bound
	if err := fault.Clock(time.Hour).Apply(ctx); err != nil {
		t.Fatalf("Apply returned %v, want nil", err)
	}
	if got := fault.Skew(ctx); got != 0 {
		t.Fatalf("Skew = %v, want 0 (no cell)", got)
	}
}

func TestClockConcurrent(t *testing.T) {
	ctx := fault.WithClock(context.Background())
	f := fault.Clock(time.Minute)
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = f.Apply(ctx) }()
		go func() { defer wg.Done(); _ = fault.Skew(ctx) }()
	}
	wg.Wait()
	if got := fault.Skew(ctx); got != time.Minute {
		t.Fatalf("Skew = %v, want 1m", got)
	}
}
