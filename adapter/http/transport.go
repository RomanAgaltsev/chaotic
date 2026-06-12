//go:build !chaos_off

// Package http wraps an http.RoundTripper so outbound client calls are
// subject to chaos. Construct an engine, attach rules, then pass
// http.DefaultTransport (or any RoundTripper) and the engine to WrapTransport.
// The returned RoundTripper is safe to share across goroutines.
package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
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
		var hse *fault.HTTPStatusError
		if errors.As(err, &hse) {
			resp := synthResponse(req, hse)
			var callErr error
			if hse.Code >= 500 {
				callErr = fmt.Errorf("chaotic: synthesized %d", hse.Code)
			}
			reportOutcome(ctx, action, callErr)
			_ = action.After(ctx)
			return resp, nil
		}
		var hf *fault.HeaderFault
		if errors.As(err, &hf) {
			// A header fault mutates the response the caller reads, then the
			// real call proceeds normally.
			resp, rerr := t.wrapped.RoundTrip(req)
			if resp != nil {
				applyHeaderFault(resp.Header, hf)
			}
			reportOutcome(ctx, action, rerr)
			_ = action.After(ctx)
			return resp, rerr
		}
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

// synthResponse builds a real *http.Response carrying the fault's status, so
// client retry/handling code can be exercised against e.g. a 503 instead of a
// transport error. Body defaults to http.StatusText(code).
func synthResponse(req *http.Request, hse *fault.HTTPStatusError) *http.Response {
	body := hse.Body
	if body == "" {
		body = http.StatusText(hse.Code)
	}
	return &http.Response{
		StatusCode:    hse.Code,
		Status:        strconv.Itoa(hse.Code) + " " + http.StatusText(hse.Code),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

// applyHeaderFault mutates h per the header fault: delete on strip, set otherwise.
func applyHeaderFault(h http.Header, hf *fault.HeaderFault) {
	if hf.Strip {
		h.Del(hf.Key)
		return
	}
	h.Set(hf.Key, hf.Value)
}
