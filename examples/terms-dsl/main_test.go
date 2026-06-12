package main

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/chaos"
	"github.com/RomanAgaltsev/chaotic/engine"
)

func TestOneLinerActivatesChaos(t *testing.T) {
	eng := engine.New()
	ctx := chaos.WithEngine(context.Background(), eng)

	// "fail the first two checkout points, then recover" — one line, no Go rules.
	if err := Activate(eng, `checkout: kind(explicit),name(checkout)=2*error("payment down")`); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	results := []error{
		chaos.Point(ctx, "checkout"),
		chaos.Point(ctx, "checkout"),
		chaos.Point(ctx, "checkout"),
	}
	if results[0] == nil || results[1] == nil {
		t.Fatalf("first two points should fail, got %v, %v", results[0], results[1])
	}
	if results[2] != nil {
		t.Fatalf("third point should pass (Times(2) exhausted), got %v", results[2])
	}
	if got := eng.Hits("checkout"); got != 2 {
		t.Fatalf("rule fired %d times, want 2", got)
	}
}
