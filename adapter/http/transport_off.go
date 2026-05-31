//go:build chaos_off

// Package http (chaos_off build): WrapTransport is no-op passthrough.
package http

import (
	"net/http"

	"github.com/ag4r/chaotic/engine"
)

// WrapTransport returns rt unchanged under the chaos_off build tag
// (or http.DefaultTransport when rt is nil). The engine is ignored.
func WrapTransport(rt http.RoundTripper, eng *engine.Engine) http.RoundTripper {
	if rt == nil {
		return http.DefaultTransport
	}
	return rt
}
