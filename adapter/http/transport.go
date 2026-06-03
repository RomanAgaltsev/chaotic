//go:build !chaos_off

// Package http wraps an http.RoundTripper so outbound client calls are
// subject to chaos. Construct an engine, attach rules, then pass
// http.DefaultTransport (or any RoundTripper) and the engine to WrapTransport.
// The returned RoundTripper is safe to share across goroutines.
package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"syscall"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// WrapTransport returns a RoundTripper that consults eng on every request.
// If eng is nil or has no rules, the wrapper is a near-zero-cost passthrough.
func WrapTransport(rt http.RoundTripper, eng *engine.Engine) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &transport{
		wrapped: rt,
		eng:     eng,
	}
}

type transport struct {
	wrapped http.RoundTripper
	eng     *engine.Engine
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.eng.Enabled() {
		return t.wrapped.RoundTrip(req)
	}
	ctx := req.Context()
	op := engine.Op{
		Kind:   engine.OpHTTPClient,
		Name:   req.URL.Path,
		Method: req.Method,
		Attrs: map[string]string{
			"host": req.URL.Host,
		},
	}
	action := t.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		// The injected fault aborted the call. Report it as the outcome so the
		// failure budget counts injected errors, then run After to release any
		// held bound (e.g. the max-concurrent slot).
		reportOutcome(ctx, action, err)
		_ = action.After(ctx)
		return nil, translateError(req.URL, err)
	}
	resp, err := t.wrapped.RoundTrip(req)
	reportOutcome(ctx, action, err)
	_ = action.After(ctx)
	return resp, err
}

// reportOutcome forwards the wrapped call's error (or the injected fault) to the
// engine if the action supports outcome reporting.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
}

func translateError(u *url.URL, err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.ECONNRESET,
		}
	}
	return &url.Error{
		Op:  "chaos",
		URL: u.String(),
		Err: err,
	}
}
