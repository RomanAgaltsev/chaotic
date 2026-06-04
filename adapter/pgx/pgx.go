package pgx

import (
	"context"

	"github.com/ag4r/chaotic/engine"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// finish reports the call's outcome to the failure budget (when the action
// supports it) and releases any held bound — notably a WithMaxConcurrent slot —
// by running After. Every intercepted method must call finish exactly once per
// fired action, or the slot leaks and the budget never sees the call. A nil
// action is a no-op.
func finish(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}

// WrapPool returns a *Pool that proxies *pgxpool.Pool through the chaotic
// engine. A nil engine is a programmer error and panics at wrap time.
func WrapPool(p *pgxpool.Pool, eng *engine.Engine) *Pool {
	if eng == nil {
		panic("adapter/pgx: WrapPool requires a non-nil *engine.Engine")
	}
	return &Pool{
		b:   p,
		eng: eng,
		raw: p,
	}
}

// WrapConn returns a *Conn that proxies a standalone *pgx.Conn through the
// chaotic engine. A nil engine is a programmer error and panics at wrap time.
func WrapConn(c *pgxv5.Conn, eng *engine.Engine) *Conn {
	if eng == nil {
		panic("adapter/pgx: WrapConn requires a non-nil *engine.Engine")
	}
	return &Conn{
		b:   &standaloneConnBackend{Conn: c},
		eng: eng,
		raw: c,
	}
}
