//go:build !chaos_off

package aws

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/middleware"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each request, or chaos silently
// stops once the cap is reached. fault.Latency(0) returns nil from Before, so the
// slot is freed only by HandleFinalize running After.
func TestMiddlewareReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpAWS),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	mw := chaosMiddleware{eng: eng}
	next := middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
		return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
	})
	ctx := context.Background()
	for range 3 {
		if _, _, err := mw.HandleFinalize(ctx, middleware.FinalizeInput{}, next); err != nil {
			t.Fatalf("HandleFinalize err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential requests; the max-concurrent slot is leaking", got)
	}
}
