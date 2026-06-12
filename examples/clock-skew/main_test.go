package main

import (
	"context"
	"testing"
	"time"

	"github.com/RomanAgaltsev/chaotic/chaos"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestRunShowsClockSkewExpiry(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.WithFault(fault.Clock(2*time.Hour)),
		engine.MatchName("token.validate"),
	))
	ctx := chaos.WithEngine(context.Background(), eng)

	before, after := run(ctx)
	if !before {
		t.Fatal("token should be valid before skew")
	}
	if after {
		t.Fatal("token should be expired after +2h skew past its 1h TTL")
	}
}
