//go:build !chaos_off

package chaos_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ag4r/chaotic/chaos"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExamplePoint() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.MatchName("checkout.afterCommit"),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("downstream timeout"))),
	).Named("checkout"))
	ctx := chaos.WithEngine(context.Background(), eng)

	fmt.Println("attempt 1:", chaos.Point(ctx, "checkout.afterCommit"))
	fmt.Println("attempt 2:", chaos.Point(ctx, "checkout.afterCommit"))
	// A Point on a context with no engine bound is always nil (inert in prod).
	fmt.Println("unbound:", chaos.Point(context.Background(), "checkout.afterCommit"))
	// Output:
	// attempt 1: downstream timeout
	// attempt 2: <nil>
	// unbound: <nil>
}

func ExamplePointWith() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.MatchAttr("tier", "premium"),
		engine.WithFault(fault.Error(errors.New("degraded"))),
	).Named("tiered"))
	ctx := chaos.WithEngine(context.Background(), eng)

	fmt.Println("premium:", chaos.PointWith(ctx, "lookup", map[string]string{"tier": "premium"}))
	fmt.Println("free:", chaos.PointWith(ctx, "lookup", map[string]string{"tier": "free"}))
	// Output:
	// premium: degraded
	// free: <nil>
}

// ExampleWithEngine_clock shows fault.Clock skewing engine.Now: a rule jumps
// the clock 48h ahead when an explicit point fires, so a deadline computed
// after the jump is far in the future relative to one computed before it.
func ExampleWithEngine_clock() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.WithFault(fault.Clock(48*time.Hour)),
		engine.MatchName("expiry.check"),
	))
	ctx := chaos.WithEngine(context.Background(), eng)

	start := engine.Now(ctx)             // real clock (no skew yet)
	_ = chaos.Point(ctx, "expiry.check") // fire Clock: +48h
	jumped := engine.Now(ctx)

	fmt.Println(jumped.Sub(start) > 47*time.Hour)
	// Output: true
}
