package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/RomanAgaltsev/chaotic/fault"
)

func driveFault(t *testing.T, e *Engine, callErr error) {
	t.Helper()
	ctx := context.Background()
	a := e.Eval(ctx, Op{Kind: OpHTTPClient})
	_ = a.Before(ctx)
	if o, ok := a.(OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = a.After(ctx)
}

func TestFailureBudgetSuppressesWhenOverBudget(t *testing.T) {
	boom := errors.New("downstream down")
	e := New(WithFailureBudget(0.5, 4)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	for range 4 { // fill window with 100% errors
		driveFault(t, e, boom)
	}
	if got := e.Hits("r"); got != 4 {
		t.Fatalf("Hits before tripping = %d, want 4", got)
	}
	// Window full at 100% >= 50% budget -> next call must not fire.
	driveFault(t, e, boom)
	if got := e.Hits("r"); got != 4 {
		t.Fatalf("rule fired despite over-budget: Hits = %d, want still 4", got)
	}
}

func TestFailureBudgetDoesNotSuppressBeforeWindowFull(t *testing.T) {
	boom := errors.New("x")
	e := New(WithFailureBudget(0.5, 10)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	for range 5 {
		driveFault(t, e, boom)
	}
	if got := e.Hits("r"); got != 5 {
		t.Fatalf("Hits = %d, want 5 (no suppression before window full)", got)
	}
}

func TestFailureBudgetEmitsSkipReason(t *testing.T) {
	obs := &reasonObserver{}
	boom := errors.New("x")
	e := New(WithObserver(obs), WithFailureBudget(0.5, 2)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("r"))
	driveFault(t, e, boom)
	driveFault(t, e, boom) // fills window at 100%
	driveFault(t, e, boom) // suppressed
	if len(obs.reasons) == 0 || obs.reasons[len(obs.reasons)-1] != ReasonFailureBudget {
		t.Fatalf("want last skip reason %q, got %v", ReasonFailureBudget, obs.reasons)
	}
}

// reasonObserver captures skip reasons.
type reasonObserver struct {
	reasons []string
}

func (o *reasonObserver) RuleFired(string, Op, Action) {}

func (o *reasonObserver) RuleSkipped(_ string, _ Op, r string) {
	o.reasons = append(o.reasons, r)
}
