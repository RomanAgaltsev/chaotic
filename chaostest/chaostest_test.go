package chaostest_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ag4r/chaotic/chaostest"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestNewReturnsEngineBoundToCleanup(t *testing.T) {
	// We can't directly observe t.Cleanup, but we can ensure New returns
	// a usable engine and that calling it twice gives two independent engines.
	a := chaostest.New(t)
	b := chaostest.New(t)
	a.AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("a")))).Named("a"))
	if !a.Enabled() || b.Enabled() {
		t.Fatalf("engines not independent: a.Enabled=%v b.Enabled=%v", a.Enabled(), b.Enabled())
	}
}

func TestNewEnginesIsolatedFromMainEngine(t *testing.T) {
	// Repeated calls to chaostest.New produce engines that don't share state.
	e1 := chaostest.New(t)
	e2 := chaostest.New(t)
	e1.AddRule(engine.NewRule().Named("only-in-e1"))
	if h := e2.Hits("only-in-e1"); h != 0 {
		t.Fatalf("e2 leaked from e1: hits=%d", h)
	}
	if h := e1.Hits("only-in-e1"); h != 0 {
		t.Fatalf("e1 fired without Eval: hits=%d", h)
	}
	e1.Eval(context.Background(), engine.Op{})
	if h := e1.Hits("only-in-e1"); h != 1 {
		t.Fatalf("e1.Hits = %d, want 1", h)
	}
}

// fakeTB captures Errorf calls so we can assert that helpers fail correctly.
type fakeTB struct {
	testing.TB
	errors []string
}

func (f *fakeTB) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeTB) Helper() {}

func TestAssertHitsPassesOnMatch(t *testing.T) {
	e := chaostest.New(t)
	e.AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("r"))
	e.Eval(context.Background(), engine.Op{})
	e.Eval(context.Background(), engine.Op{})
	ft := &fakeTB{}
	chaostest.AssertHits(ft, e, "r", 2)
	if len(ft.errors) != 0 {
		t.Fatalf("AssertHits reported errors: %v", ft.errors)
	}
}

func TestAssertHitsFailsOnMismatch(t *testing.T) {
	e := chaostest.New(t)
	e.AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("r"))
	ft := &fakeTB{}
	chaostest.AssertHits(ft, e, "r", 3)
	if len(ft.errors) == 0 {
		t.Fatal("AssertHits did not report a mismatch")
	}
}

func TestAssertEventsExhaustedRequiresAllNamedRulesNonZero(t *testing.T) {
	e := chaostest.New(t)
	e.AddRule(engine.NewRule(engine.WithFault(fault.Error(errors.New("x")))).Named("a"))
	e.AddRule(engine.NewRule(engine.MatchKind(engine.OpSQL), engine.WithFault(fault.Error(errors.New("y")))).Named("b"))
	e.Eval(context.Background(), engine.Op{}) // fires "a", not "b"
	ft := &fakeTB{}
	chaostest.AssertEventsExhausted(ft, e)
	if len(ft.errors) == 0 {
		t.Fatal("expected AssertEventsExhausted to flag 'b' as never fired")
	}
	if !strings.Contains(ft.errors[0], "b") {
		t.Fatalf("error %q should mention 'b'", ft.errors[0])
	}
}
