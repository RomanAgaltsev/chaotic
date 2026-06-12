//go:build !chaos_off

package aws

import (
	"context"
	"errors"
	"io"
	"net"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// chaosMiddlewareID is the smithy middleware ID under which the chaos is
// registered. Exported-ish via this constant so tests (and de-registration) can
// find it.
const chaosMiddlewareID = "ChaoticChaos"

// Step selects which smithy stack the chaos middleware registers on.
type Step int

const (
	// StepFinalize (default) runs after retry classification is set up and before
	// send, so an injected error is retried by the SDK exactly like a real failure.
	StepFinalize Step = iota
	// StepBuild runs before signing and retry classification.
	StepBuild
)

// MiddlewareOptions configures AppendChaosMiddlewareWith. The zero value selects
// StepFinalize.
type MiddlewareOptions struct {
	Step Step
}

// AppendChaosMiddleware registers the chaos middleware on cfg at the Finalize step
// (the pinned default). Any service client built from cfg becomes chaos-aware.
func AppendChaosMiddleware(cfg *awssdk.Config, eng *engine.Engine) {
	AppendChaosMiddlewareWith(cfg, eng, MiddlewareOptions{})
}

// AppendChaosMiddlewareWith is AppendChaosMiddleware with an explicit step choice.
func AppendChaosMiddlewareWith(cfg *awssdk.Config, eng *engine.Engine, opts MiddlewareOptions) {
	mw := chaosMiddleware{eng: eng}
	cfg.APIOptions = append(cfg.APIOptions, func(stack *middleware.Stack) error {
		switch opts.Step {
		case StepBuild:
			return stack.Build.Add(mw, middleware.After)
		default:
			return stack.Finalize.Add(mw, middleware.After)
		}
	})
}

// chaosMiddleware implements both the Finalize and Build smithy middleware
// interfaces so it can register on either step. Only the registered step's Handle
// method is invoked per request.
type chaosMiddleware struct {
	eng *engine.Engine
}

func (chaosMiddleware) ID() string {
	return chaosMiddlewareID
}

// HandleFinalize is the Finalize-step entry point (the default).
func (m chaosMiddleware) HandleFinalize(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
	if !m.eng.Enabled() {
		return next.HandleFinalize(ctx, in)
	}
	action := m.eng.Eval(ctx, m.op(ctx))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return middleware.FinalizeOutput{}, middleware.Metadata{}, mapErr(err)
	}
	out, md, err := next.HandleFinalize(ctx, in)
	reportOutcome(ctx, action, err)
	return out, md, err
}

// HandleBuild is the Build-step entry point (opt-in via StepBuild).
func (m chaosMiddleware) HandleBuild(ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
	if !m.eng.Enabled() {
		return next.HandleBuild(ctx, in)
	}
	action := m.eng.Eval(ctx, m.op(ctx))
	if err := action.Before(ctx); err != nil {
		reportOutcome(ctx, action, err)
		return middleware.BuildOutput{}, middleware.Metadata{}, mapErr(err)
	}
	out, md, err := next.HandleBuild(ctx, in)
	reportOutcome(ctx, action, err)
	return out, md, err
}

// op builds the Op from request-context metadata only (never the payload).
func (chaosMiddleware) op(ctx context.Context) engine.Op {
	service := awsmiddleware.GetServiceID(ctx)
	operation := awsmiddleware.GetOperationName(ctx)
	region := awsmiddleware.GetRegion(ctx)
	return engine.Op{
		Kind:   engine.OpAWS,
		Name:   service + "." + operation,
		Method: "request",
		Attrs: map[string]string{
			"service":   service,
			"operation": operation,
			"region":    region,
		},
	}
}

// mapErr translates a fault error into a model the SDK retryer understands.
// ConnDrop becomes a *net.OpError with Op "dial" wrapping io.ErrUnexpectedEOF.
// The SDK's RetryableConnectionError classifier treats a *net.OpError as
// retryable only when Op == "dial" or the wrapped error reports Temporary()==true
// (see aws/retry: a "read" op wrapping a non-temporary error such as
// io.ErrUnexpectedEOF is NOT retried). Op "dial" makes the injected drop retryable
// on every platform without depending on a syscall constant. Every other fault
// error passes through unchanged, so a caller who wants a specific API error
// supplies fault.Error(&smithy.GenericAPIError{...}).
func mapErr(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return &net.OpError{Op: "dial", Net: "tcp", Err: io.ErrUnexpectedEOF}
	}
	return err
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// when the action reports outcomes, then runs After to release any held bound
// (e.g. a WithMaxConcurrent slot). Call it exactly once per action, or the slot
// leaks and the failure budget never sees the call. A nil action is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}
