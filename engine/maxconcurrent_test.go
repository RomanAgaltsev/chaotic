package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

func TestMaxConcurrentCapsInFlight(t *testing.T) {
	e := New(WithMaxConcurrent(1)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	ctx := context.Background()
	a1 := e.Eval(ctx, Op{Kind: OpHTTPClient}) // acquires the only slot, fires
	e.Eval(ctx, Op{Kind: OpHTTPClient})       // no slot -> capped
	if got := e.Hits("r"); got != 1 {
		t.Fatalf("Hits = %d, want 1 (second call capped)", got)
	}
	_ = a1.After(ctx) // releases the slot
	a3 := e.Eval(ctx, Op{Kind: OpHTTPClient})
	if got := e.Hits("r"); got != 2 {
		t.Fatalf("Hits = %d, want 2 after slot released", got)
	}
	_ = a3.After(ctx)
}

func TestMaxConcurrentReleasesOnBeforeShortCircuit(t *testing.T) {
	e := New(WithMaxConcurrent(1)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(errors.New("x")))).Named("r"))
	ctx := context.Background()
	a1 := e.Eval(ctx, Op{Kind: OpHTTPClient})
	_ = a1.Before(ctx) // error fault short-circuits -> releases slot
	e.Eval(ctx, Op{Kind: OpHTTPClient})
	if got := e.Hits("r"); got != 2 {
		t.Fatalf("Hits = %d, want 2 (slot released after Before short-circuit)", got)
	}
}
