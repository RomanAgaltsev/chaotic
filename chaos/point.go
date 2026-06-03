//go:build !chaos_off

// Package chaos provides explicit injection points: chaos.Point(ctx, name)
// consults the engine bound to ctx and returns a fault error (or nil) at a
// place no adapter wraps - between two pure functions, inside a goroutine, at a
// state-machine transition.
//
// Bind an engine once near a test or request boundary:
//
//	ctx = chaos.WithEngine(ctx, eng)
//
// then place points anywhere downstream:
//
//	if err := chaos.Point(ctx, "checkout.afterCommit"); err != nil {
//		return err
//	}
//
// A Point on a context with no engine bound (or a disabled engine) is a silent,
// allocation-free no-op, so points are safe to leave in production code.
// Recommended naming is dotted and hierarchical ("checkout.afterCommit").
// Names are free-form and matched by engine rules via MatchName.
package chaos

import (
	"context"

	"github.com/ag4r/chaotic/engine"
)

type engineKey struct{}

// WithEngine returns a child context carrying eng. Point and PointWith consult
// the engine bound to the context they are given. Binding nil clears the engine
// for a sub-scope (Point becomes a no-op there).
func WithEngine(ctx context.Context, eng *engine.Engine) context.Context {
	return context.WithValue(ctx, engineKey{}, eng)
}

// engineFrom returns the engine bound to ctx, or nil if none.
func engineFrom(ctx context.Context) *engine.Engine {
	eng, _ := ctx.Value(engineKey{}).(*engine.Engine)
	return eng
}

// Point consults the engine bound to ctx and returns either nil (no engine
// bound, chaos disabled, or no rule matched) or the fault error produced by a
// matching rule. Latency faults sleep inline before Point returns. A Panic
// fault panics.
func Point(ctx context.Context, name string) error {
	return PointWith(ctx, name, nil)
}

// PointWith is Point with an attribute bag, mirroring engine.Op.Attrs. Rules may
// match on these attrs via engine.MatchAttr. attrs may be nil.
func PointWith(ctx context.Context, name string, attrs map[string]string) error {
	eng := engineFrom(ctx)
	if !eng.Enabled() { // nil-safe, alloc-free no-op path
		return nil
	}
	op := engine.Op{Kind: engine.OpExplicit, Name: name, Attrs: attrs}
	act := eng.Eval(ctx, op)
	err := act.Before(ctx)
	_ = act.After(ctx)
	if or, ok := act.(engine.OutcomeReporter); ok {
		or.Outcome(ctx, err)
	}
	return err
}
