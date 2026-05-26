package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ag4r/chaotic/fault"
)

func TestNewReturnsEmptyEngine(t *testing.T) {
	e := New()
	if e.Enabled() {
		t.Fatal("freshly constructed engine should not be Enabled")
	}
}

func TestAddRuleEnablesEngine(t *testing.T) {
	e := New().AddRule(NewRule().Named("r"))
	if !e.Enabled() {
		t.Fatal("engine with one rule should be Enabled")
	}
}

func TestAddRuleIsCnainable(t *testing.T) {
	e := New().AddRule(NewRule().Named("a")).AddRule(NewRule().Named("b"))
	if got := e.rules.Load().Len(); got != 2 {
		t.Fatalf("rule count = %d, want 2", got)
	}
}

func TestResetClearsRulesAndCounters(t *testing.T) {
	e := New().AddRule(NewRule().Named("r"))
	e.Reset()
	if e.Enabled() {
		t.Fatal("engine after Reset should not be Enabled")
	}
	if got := len(e.AllHits()); got != 0 {
		t.Fatalf("AllHits len = %d, want 0", got)
	}
}

func TestAddRuleIsConcurrencySafe(t *testing.T) {
	e := New()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.AddRule(NewRule().Named("r"))
		}()
	}
	wg.Wait()
	if got := e.rules.Load().Len(); got != 50 {
		t.Fatalf("rule count = %d, want 50", got)
	}
}

func TestEnabledIsNilSafe(t *testing.T) {
	var e *Engine
	if e.Enabled() {
		t.Fatal("nil engine should not be Enabled")
	}
}

func TestEvalReturnsPassWhenNoRules(t *testing.T) {
	e := New()
	a := e.Eval(context.Background(), Op{Kind: OpHTTPClient})
	if a != Pass {
		t.Fatalf("Eval returned %v, want Pass", a)
	}
}

func TestEvalReturnsActionWhenRuleMatches(t *testing.T) {
	target := errors.New("boom")
	e := New().AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(target))))
	a := e.Eval(context.Background(), Op{Kind: OpHTTPClient})
	if a == Pass {
		t.Fatal("Eval returned Pass, wanted action")
	}
	if err := a.Before(context.Background()); !errors.Is(err, target) {
		t.Fatalf("Before returned %v, want errors.Is(%v) == true", err, target)
	}
}

func TestEvalFirstMatchingRuleWins(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	e := New().
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(first)))).
		AddRule(NewRule(MatchKind(OpHTTPClient), WithFault(fault.Error(second))))
	a := e.Eval(context.Background(), Op{Kind: OpHTTPClient})
	if err := a.Before(context.Background()); !errors.Is(err, first) {
		t.Fatalf("Before returned %v, want %v (first rule wins)", err, first)
	}

}

func TestEvalSkipsRulesWhoseCounterRefuses(t *testing.T) {
	target := errors.New("boom")
	e := New().
		AddRule(NewRule(MatchKind(OpHTTPClient), Times(1), WithFault(fault.Error(target))))
	// First call fires.
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient}); a == Pass {
		t.Fatal("first call returned Pass, want action")
	}
	// Second call should be skipped.
	if a := e.Eval(context.Background(), Op{Kind: OpHTTPClient}); a != Pass {
		t.Fatal("second call did not return Pass")
	}
}

func TestEvalRunsKillSwitch(t *testing.T) {
	target := errors.New("boom")
	e := New(WithKillSwitch(func(_ context.Context, _ Op) bool { return true })).
		AddRule(NewRule(WithFault(fault.Error(target))))
	if a := e.Eval(context.Background(), Op{}); a != Pass {
		t.Fatal("kill switch did not short-circuit")
	}
}

func TestEvalNilEngineReturnsPass(t *testing.T) {
	var e *Engine
	if a := e.Eval(context.Background(), Op{}); a != Pass {
		t.Fatal("nil engine should return Pass")
	}
}

func TestActionBeforeRunsFaultsInOrder(t *testing.T) {
	var seq []int
	track := func(n int) fault.Fault {
		return faultFn(func(context.Context) error {
			seq = append(seq, n)
			return nil
		})
	}
	e := New().AddRule(NewRule(WithFaults(track(1), track(2), track(3))))
	a := e.Eval(context.Background(), Op{})
	if err := a.Before(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(seq) != 3 || seq[0] != 1 || seq[1] != 2 || seq[2] != 3 {
		t.Fatalf("sequence = %v, want [1 2 3]", seq)
	}
}

// faultFn is a tiny test helper that lets us treat any function as a Fault.
type faultFn func(context.Context) error

func (f faultFn) Apply(ctx context.Context) error {
	return f(ctx)
}

func TestActionBeforeShortCircuitsOnFirstError(t *testing.T) {
	var seq []int
	stop := errors.New("stop")
	track := func(n int, err error) fault.Fault {
		return faultFn(func(context.Context) error {
			seq = append(seq, n)
			return err
		})
	}
	e := New().AddRule(NewRule(WithFaults(track(1, nil), track(2, stop), track(3, nil))))
	a := e.Eval(context.Background(), Op{})
	err := a.Before(context.Background())
	if !errors.Is(err, stop) {
		t.Fatalf("Before returned %v, want %v", err, stop)
	}
	if len(seq) != 2 {
		t.Fatalf("sequence = %v, want first two only", seq)
	}
}

func TestHitsIncrementsOnFire(t *testing.T) {
	target := errors.New("boom")
	e := New().AddRule(NewRule(WithFault(fault.Error(target))).Named("r"))
	for range 3 {
		e.Eval(context.Background(), Op{})
	}
	if got := e.Hits("r"); got != 3 {
		t.Fatalf("Hits(r) = %d, want 3", got)
	}
}

func TestHitsDoesNotIncrementForUnnamedRule(t *testing.T) {
	target := errors.New("boom")
	e := New().AddRule(NewRule(WithFault(fault.Error(target))))
	e.Eval(context.Background(), Op{})
	if got := e.Hits(""); got != 0 {
		t.Fatalf("Hits(\"\") = %d, want 0", got)
	}
}

func TestHitsUnknownReturnsZero(t *testing.T) {
	e := New()
	if got := e.Hits("nope"); got != 0 {
		t.Fatalf("Hits(nope) = %d, want 0", got)
	}
}

func TestAllHitsSnapshot(t *testing.T) {
	target := errors.New("boom")
	e := New().
		AddRule(NewRule(WithFault(fault.Error(target))).Named("a")).
		AddRule(NewRule(MatchKind(OpSQL), WithFault(fault.Error(target))).Named("b"))
	// "a" fires twice, "b" fires once
	e.Eval(context.Background(), Op{})
	e.Eval(context.Background(), Op{})
	e.Eval(context.Background(), Op{Kind: OpSQL})
	// Note: "a" has no kind filter so it also matches the OpSQL call.
	// So "a" = 3, "b" = 0.
	snapshot := e.AllHits()
	if snapshot["a"] != 3 {
		t.Errorf("a = %d, want 3", snapshot["a"])
	}
	if snapshot["b"] != 0 {
		t.Errorf("b = %d, want 0", snapshot["b"])
	}
}

func TestHitsCountsAreRaceFree(t *testing.T) {
	target := errors.New("boom")
	e := New().AddRule(NewRule(WithFault(fault.Error(target))).Named("r"))
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Eval(context.Background(), Op{})
		}()
	}
	wg.Wait()
	if got := e.Hits("r"); got != 100 {
		t.Fatalf("Hits(r) = %d, want 100", got)
	}
}
