package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

func TestRateLimitCapsFires(t *testing.T) {
	e := New(WithRateLimit(1)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	ctx := context.Background()
	e.Eval(ctx, Op{Kind: OpHTTPClient}) // consumes the only token -> fires
	e.Eval(ctx, Op{Kind: OpHTTPClient}) // no token yet -> skipped
	if got := e.Hits("r"); got != 1 {
		t.Fatalf("Hits = %d, want 1 (second call rate-limited)", got)
	}
}

func TestRateLimitEmitsSkipReason(t *testing.T) {
	obs := &reasonObserver{}
	e := New(WithObserver(obs), WithRateLimit(1)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	ctx := context.Background()
	e.Eval(ctx, Op{Kind: OpHTTPClient})
	e.Eval(ctx, Op{Kind: OpHTTPClient})
	if len(obs.reasons) != 1 || obs.reasons[0] != ReasonRateLimit {
		t.Fatalf("want one skip with reason %q, got %v", ReasonRateLimit, obs.reasons)
	}
	_ = errors.New // keep imports tidy if unused elsewhere
}
