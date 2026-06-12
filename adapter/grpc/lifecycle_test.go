//go:build !chaos_off

package grpc_test

import (
	"context"
	"testing"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each unary call, or chaos
// silently stops once the cap is reached. The latency fault returns nil from
// Before, so the slot is freed only by the interceptor running After.
func TestUnaryClientInterceptorReleasesMaxConcurrentSlot(t *testing.T) {
	e := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))
	intercept := chaosgrpc.UnaryClientInterceptor(e)
	inv := &nopInvoker{}
	for range 3 {
		if err := intercept(context.Background(), "/svc/Method", nil, nil, nil, inv.invoke); err != nil {
			t.Fatalf("intercept err = %v", err)
		}
	}
	if got := e.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential calls; the max-concurrent slot is leaking", got)
	}
}
