// Package httpsrv provides net/http middleware that subjects inbound requests
// to chaos. The Middleware mounts in any net/http chain.
package httpsrv

import (
	"errors"
	"net/http"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
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
				handleErr(w, err)
				return
			}
			next.ServeHTTP(w, r)
			_ = action.After(ctx)
		})
	}
}

func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, fault.ErrConnDrop) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "chaos: conn drop (no hijacker)", http.StatusInternalServerError)
			return
		}
		conn, _, herr := hj.Hijack()
		if herr != nil {
			http.Error(w, "chaos: conn drop (hijack failed)", http.StatusInternalServerError)
			return
		}
		_ = conn.Close()
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
