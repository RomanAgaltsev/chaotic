//go:build !chaos_off

package aws

import (
	"context"
	"errors"
	"net"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// finalizeNextOK is a stub Finalize handler that "succeeds" without a network call.
func finalizeNextOK(called *bool) middleware.FinalizeHandler {
	return middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
		*called = true
		return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
	})
}

func TestMiddlewareInjectsError(t *testing.T) {
	sentinel := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpAWS),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("err"))

	mw := chaosMiddleware{eng: eng}
	called := false
	_, _, err := mw.HandleFinalize(context.Background(), middleware.FinalizeInput{}, finalizeNextOK(&called))

	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if called {
		t.Fatal("next handler ran despite the fault (should short-circuit)")
	}
}

func TestMiddlewareConnDropMapsToNetOpError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpAWS),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("drop"))

	mw := chaosMiddleware{eng: eng}
	called := false
	_, _, err := mw.HandleFinalize(context.Background(), middleware.FinalizeInput{}, finalizeNextOK(&called))

	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
}

func TestMiddlewarePassesThroughWhenNoRule(t *testing.T) {
	mw := chaosMiddleware{eng: engine.New()} // no rules
	called := false
	_, _, err := mw.HandleFinalize(context.Background(), middleware.FinalizeInput{}, finalizeNextOK(&called))
	if err != nil {
		t.Fatalf("err = %v, want nil passthrough", err)
	}
	if !called {
		t.Fatal("next handler was not called on the no-rule path")
	}
}

func TestAppendChaosMiddlewareRegistersOnFinalize(t *testing.T) {
	var cfg awssdk.Config
	AppendChaosMiddleware(&cfg, engine.New())
	if len(cfg.APIOptions) != 1 {
		t.Fatalf("APIOptions len = %d, want 1", len(cfg.APIOptions))
	}
	stack := middleware.NewStack("test", smithyhttp.NewStackRequest)
	if err := cfg.APIOptions[0](stack); err != nil {
		t.Fatalf("APIOption returned err: %v", err)
	}
	if _, ok := stack.Finalize.Get(chaosMiddlewareID); !ok {
		t.Fatalf("chaos middleware %q not registered on the Finalize step", chaosMiddlewareID)
	}
}
