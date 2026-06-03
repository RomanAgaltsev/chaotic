package chaos_test

import (
	"context"
	"errors"
	"fmt"

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
