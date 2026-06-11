package engine

import (
	"context"
	"testing"
)

func TestWithPerRuleRateLimitThrottlesOneRule(t *testing.T) {
	// rps=1, burst=1: of two back-to-back matches, only the first fires.
	eng := New().AddRule(NewRule(
		MatchKind(OpHTTPClient),
		Always(),
		WithPerRuleRateLimit(1),
		WithFault(errFaultForTest()),
	).Named("limited"))

	ctx := context.Background()
	op := Op{Kind: OpHTTPClient}

	first := eng.Eval(ctx, op)
	second := eng.Eval(ctx, op)

	if _, ok := first.(*ruleAction); !ok {
		t.Fatal("first call should fire (token available)")
	}
	if second != Pass {
		t.Fatal("second back-to-back call should be throttled to Pass")
	}
	if got := eng.Hits("limited"); got != 1 {
		t.Fatalf("rule fired %d times, want 1 (one token)", got)
	}
}
