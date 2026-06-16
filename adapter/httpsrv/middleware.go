//go:build !chaos_off

// Package httpsrv provides net/http middleware that subjects inbound requests
// to chaos. The Middleware mounts in any net/http chain.
package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// Middleware returns an http middleware that consults eng on every request.
// If eng is nil or has no rules, the wrapper is a near-zero-cost passthrough.
func Middleware(eng *engine.Engine) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !eng.Enabled() {
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			op := engine.Op{
				Kind:   engine.OpHTTPServer,
				Name:   r.URL.Path,
				Method: r.Method,
				Attrs: map[string]string{
					"remote": r.RemoteAddr,
				},
			}
			action := eng.Eval(ctx, op)
			if err := action.Before(ctx); err != nil {
				var hf *fault.HeaderFaultError
				if !errors.As(err, &hf) {
					// A short-circuiting fault (error, conn drop, HTTP status)
					// aborted the request. Report the outcome, release the bound,
					// and render the error.
					reportOutcome(ctx, action, err)
					_ = action.After(ctx)
					handleErr(w, err)
					return
				}
				// A header fault mutates the request the handler sees, then the
				// request proceeds normally.
				applyHeaderFault(r.Header, hf)
			}
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			var callErr error
			if rec.status >= 500 {
				callErr = fmt.Errorf("chaotic: server responded %d", rec.status)
			}
			reportOutcome(ctx, action, callErr)
			_ = action.After(ctx)
		})
	}
}

// reportOutcome forwards the call's error (or the injected fault) to the engine
// if the action supports outcome reporting.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
}

func handleErr(w http.ResponseWriter, err error) {
	var hse *fault.HTTPStatusError
	if errors.As(err, &hse) {
		body := hse.Body
		if body == "" {
			body = http.StatusText(hse.Code)
		}
		http.Error(w, body, hse.Code)
		return
	}
	if errors.Is(err, fault.ErrConnDrop) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "chaotic: conn drop (no hijacker)", http.StatusInternalServerError)
			return
		}
		conn, _, herr := hj.Hijack()
		if herr != nil {
			http.Error(w, "chaotic: conn drop (hijack failed)", http.StatusInternalServerError)
			return
		}
		_ = conn.Close()
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// applyHeaderFault mutates h per the header fault: delete on strip, set otherwise.
func applyHeaderFault(h http.Header, hf *fault.HeaderFaultError) {
	if hf.Strip {
		h.Del(hf.Key)
		return
	}
	h.Set(hf.Key, hf.Value)
}

// statusRecorder captures the response status for outcome reporting. It exposes
// the underlying ResponseWriter via Unwrap so http.ResponseController users
// retain Flush/Hijack support.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.status = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.written = true
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
