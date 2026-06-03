package chaos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ag4r/chaotic/chaos"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestWithEngineRoundTrips(t *testing.T) {
	eng := engine.New()
	ctx := chaos.WithEngine(context.Background(), eng)
	// An unbound point is a silent no-op.
	// A bound-but-empty engine likewise.
	if err := chaos.Point(ctx, "p"); err != nil {
		t.Fatalf("Point on empty engine = %v, want nil", err)
	}
	if err := chaos.Point(context.Background(), "p"); err != nil {
		t.Fatalf("Point on unbound ctx = %v, want nil", err)
	}
}

func TestPointFiresErrorFault(t *testing.T) {
	target := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.MatchName("checkout.*"),
		engine.WithFault(fault.Error(target)),
	).Named("p"))
	ctx := chaos.WithEngine(context.Background(), eng)
	if err := chaos.Point(ctx, "checkout.afterCommit"); !errors.Is(err, target) {
		t.Fatalf("Point err = %v, want %v", err, target)
	}
	// Non-matching name → no fault.
	if err := chaos.Point(ctx, "login.start"); err != nil {
		t.Fatalf("non-matching Point err = %v, want nil", err)
	}
}

func TestPointLatencySleepsInline(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.WithFault(fault.Latency(40*time.Millisecond)),
	).Named("slow"))
	ctx := chaos.WithEngine(context.Background(), eng)
	start := time.Now()
	_ = chaos.Point(ctx, "anything")
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("Point returned after %v, expected to sleep ~40ms inline", elapsed)
	}
}

func TestPointPanicPropagates(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.WithFault(fault.Panic("kaboom")),
	).Named("boom"))
	ctx := chaos.WithEngine(context.Background(), eng)
	defer func() {
		if r := recover(); r != "kaboom" {
			t.Fatalf("recover = %v, want kaboom", r)
		}
	}()
	_ = chaos.Point(ctx, "x")
	t.Fatal("Point did not panic")
}

func TestPointWithAttrsMatch(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.MatchAttr("tier", "premium"),
		engine.WithFault(fault.Error(errors.New("degraded"))),
	).Named("attr"))
	ctx := chaos.WithEngine(context.Background(), eng)
	if err := chaos.PointWith(ctx, "checkout", map[string]string{"tier": "premium"}); err == nil {
		t.Fatal("expected fault for tier=premium, got nil")
	}
	if err := chaos.PointWith(ctx, "checkout", map[string]string{"tier": "free"}); err != nil {
		t.Fatalf("unexpected fault for tier=free: %v", err)
	}
}

func TestPointOutcomeFeedsFailureBudget(t *testing.T) {
	// Budget trips once the error rate over the window reaches 0.5.
	// Error points report their fault as the outcome, filling the budget.
	// Once tripped, the engine skips further fires (Point returns nil).
	eng := engine.New(engine.WithFailureBudget(0.5, 4)).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpExplicit),
			engine.WithFault(fault.Error(errors.New("x"))),
		).Named("p"))
	ctx := chaos.WithEngine(context.Background(), eng)
	fires := 0
	for range 20 {
		if err := chaos.Point(ctx, "p"); err != nil {
			fires++
		}
	}
	if fires == 0 {
		t.Fatal("no points fired; budget should allow some before tripping")
	}
	if fires == 20 {
		t.Fatal("every point fired; failure budget never engaged")
	}
}

type countingObserver struct {
	fired int
}

func (o *countingObserver) RuleFired(string, engine.Op, engine.Action) {
	o.fired++
}

func (o *countingObserver) RuleSkipped(string, engine.Op, string) {}

func TestPointEmitsObserverEvent(t *testing.T) {
	obs := &countingObserver{}
	eng := engine.New(engine.WithObserver(obs)).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpExplicit),
			engine.WithFault(fault.Latency(0)),
		).Named("p"))
	ctx := chaos.WithEngine(context.Background(), eng)
	_ = chaos.Point(ctx, "p")
	if obs.fired != 1 {
		t.Fatalf("observer fired %d times, want 1", obs.fired)
	}
}
