//go:build chaos_off

// Package chaos (chaos_off build): Point/PointWith are no-op passthrough and
// WithEngine returns ctx unchanged. The engine is ignored.
package chaos

import (
	"context"

	"github.com/ag4r/chaotic/engine"
)

// WithEngine returns ctx unchanged under the chaos_off build tag.
func WithEngine(ctx context.Context, _ *engine.Engine) context.Context {
	return ctx
}

// Point is a no-op returning nil under the chaos_off build tag.
func Point(_ context.Context, _ string) error {
	return nil
}

// PointWith is a no-op returning nil under the chaos_off build tag.
func PointWith(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
