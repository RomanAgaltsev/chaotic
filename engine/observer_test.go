package engine

import (
	"context"
	"testing"
	"time"

	"github.com/ag4r/chaotic/fault"
)

type fakeObserver struct {
	fired   []string
	skipped []string
}

func (f *fakeObserver) RuleFired(named string, _ Op, _ Action) {
	f.fired = append(f.fired, named)
}

func (f *fakeObserver) RuleSkipped(named string, _ Op, _ string) {
	f.skipped = append(f.skipped, named)
}

// richFake is an Observer that also implements RichObserver.
type richFake struct {
	fakeObserver
	events []FaultEvent
}

func (r *richFake) FaultInjected(_ context.Context, ev FaultEvent) {
	r.events = append(r.events, ev)
}

func TestRichObserverReceivesLatencyFaultEvent(t *testing.T) {
	var _ RichObserver = (*richFake)(nil)

	obs := &richFake{}
	e := New(WithObserver(obs)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(7*time.Millisecond))).Named("slow"))
	ctx := context.Background()
	a := e.Eval(ctx, Op{Kind: OpHTTPClient})
	if err := a.Before(ctx); err != nil {
		t.Fatalf("Before: %v", err)
	}
	if len(obs.events) != 1 {
		t.Fatalf("got %d FaultInjected events, want 1", len(obs.events))
	}
	ev := obs.events[0]
	if ev.Rule != "slow" {
		t.Fatalf("Rule = %q, want %q", ev.Rule, "slow")
	}
	if ev.FaultKind != fault.KindLatency {
		t.Fatalf("FaultKind = %v, want KindLatency", ev.FaultKind)
	}
	if ev.Latency != 7*time.Millisecond {
		t.Fatalf("Latency = %v, want 7ms", ev.Latency)
	}
	if ev.Op.Kind != OpHTTPClient {
		t.Fatalf("Op.Kind = %v, want OpHTTPClient", ev.Op.Kind)
	}
}

func TestRichObserverNotNotifiedForShortCircuitingFault(t *testing.T) {
	obs := &richFake{}
	e := New(WithObserver(obs)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.ConnDrop())).Named("drop"))
	ctx := context.Background()
	a := e.Eval(ctx, Op{Kind: OpHTTPClient})
	_ = a.Before(ctx) // ConnDrop returns an error -> short-circuits before any emit
	if len(obs.events) != 0 {
		t.Fatalf("got %d FaultInjected events, want 0 (fault short-circuited)", len(obs.events))
	}
}

func TestPlainObserverDoesNotReceiveFaultEvents(t *testing.T) {
	// A non-rich Observer must still work; the engine must not require RichObserver.
	obs := &fakeObserver{}
	e := New(WithObserver(obs)).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Latency(0))).Named("slow"))
	ctx := context.Background()
	a := e.Eval(ctx, Op{Kind: OpHTTPClient})
	if err := a.Before(ctx); err != nil {
		t.Fatalf("Before: %v", err)
	}
	if len(obs.fired) != 1 {
		t.Fatalf("fired = %v, want one entry", obs.fired)
	}
}

func TestObserverInterfaceCompiles(t *testing.T) {
	var _ Observer = (*fakeObserver)(nil)
}

func TestKillSwitchTypeIsCallable(t *testing.T) {
	var ks KillSwitch = func(_ context.Context, _ Op) bool {
		return true
	}
	if !ks(context.Background(), Op{}) {
		t.Fatal("kill switch returned false")
	}
}

func TestSkipReasonsAreDistinctAndNonEmpty(t *testing.T) {
	reasons := []string{
		ReasonCounter,
		ReasonRateLimit,
		ReasonMaxConcurrent,
		ReasonFailureBudget,
		ReasonDisabled,
		ReasonKillSwitch,
	}
	seen := map[string]bool{}
	for _, r := range reasons {
		if r == "" {
			t.Fatal("empty reason constant")
		}
		if seen[r] {
			t.Fatalf("duplicate reason %q", r)
		}
		seen[r] = true
	}
}
