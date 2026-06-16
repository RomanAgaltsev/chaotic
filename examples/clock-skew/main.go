// Command clock-skew demonstrates fault.Clock. A token-expiry check that reads
// the clock through engine.Now (instead of time.Now) is shown to expire a
// still-valid token once a chaos rule skews the clock past the token's TTL.
// Run with `go run .`.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/RomanAgaltsev/chaotic/chaos"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// tokenValid reports whether a token issued at issuedAt with lifetime ttl is
// still valid "now" - reading the clock through engine.Now so chaos can skew it.
func tokenValid(ctx context.Context, issuedAt time.Time, ttl time.Duration) bool {
	return engine.Now(ctx).Before(issuedAt.Add(ttl))
}

// run issues a token, checks it under the real clock, then checks it again after
// the chaos point fires a Clock fault that jumps the clock past the TTL.
func run(ctx context.Context) (before, after bool) {
	issuedAt := engine.Now(ctx) // no skew yet: real now
	const ttl = time.Hour

	before = tokenValid(ctx, issuedAt, ttl) // real clock: valid

	// A wrong/flaky server clock jumps ahead at the validation point.
	_ = chaos.Point(ctx, "token.validate")

	after = tokenValid(ctx, issuedAt, ttl) // skewed clock: expired
	return before, after
}

func main() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.WithFault(fault.Clock(2*time.Hour)),
		engine.MatchName("token.validate"),
	))
	ctx := chaos.WithEngine(context.Background(), eng)

	before, after := run(ctx)
	fmt.Fprintf(os.Stdout, "token valid before skew:    %v\n", before)
	fmt.Fprintf(os.Stdout, "token valid after +2h skew: %v\n", after)
}
