// Command chaos-point demonstrates an explicit injection point guarding a
// post-commit hook. Run with `go run .` to see the retry succeed despite an
// injected transient failure on the first attempt.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/RomanAgaltsev/chaotic/chaos"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// publish does real work, with an explicit chaos point at the spot a flaky
// downstream would fail. No adapter wraps this. The point is the boundary.
func publish(ctx context.Context, id string) error {
	if err := chaos.PointWith(ctx, "publish.afterCommit", map[string]string{"id": id}); err != nil {
		return err
	}
	return nil
}

// publishWithRetry retries publish up to attempts times.
func publishWithRetry(ctx context.Context, id string, attempts int) error {
	var err error
	for range attempts {
		if err = publish(ctx, id); err == nil {
			return nil
		}
	}
	return err
}

func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpExplicit),
		engine.MatchName("publish.afterCommit"),
		engine.Times(1), // fail only the first attempt
		engine.WithFault(fault.Error(errors.New("transient downstream error"))),
	).Named("publish-flap"))
}

func main() {
	ctx := chaos.WithEngine(context.Background(), newEngine())
	if err := publishWithRetry(ctx, "order-42", 3); err != nil {
		fmt.Fprintln(os.Stderr, "FAILED:", err)
		return
	}
	fmt.Fprintln(os.Stdout, "published after retry despite injected fault")
}
