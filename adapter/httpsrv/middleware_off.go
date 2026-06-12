//go:build chaos_off

// Package httpsrv (chaos_off build): Middleware is an identity wrapper.
package httpsrv

import (
	"net/http"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Middleware returns an identity middleware under the chaos_off build tag.
func Middleware(_ *engine.Engine) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}
