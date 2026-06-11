package fault

import (
	"context"
	"sync/atomic"
	"time"
)

// clockCell holds the wall-clock skew (in nanoseconds) observed by engine.Now
// for a context scope. It is a pointer stored in the context so a firing Clock
// fault can mutate the skew that later reads observe.
type clockCell struct {
	skew atomic.Int64
}

type clockKey struct{}

// WithClock binds a fresh, zero-skew clock cell to ctx and returns the child
// context. A Clock fault firing on this context writes its skew into the cell,
// and engine.Now reads it back. chaos.WithEngine calls this for you; call it
// directly only when using the engine without the chaos package.
func WithClock(ctx context.Context) context.Context {
	return context.WithValue(ctx, clockKey{}, &clockCell{})
}

// cellFrom returns the clock cell bound to ctx, or nil if none.
func cellFrom(ctx context.Context) *clockCell {
	c, _ := ctx.Value(clockKey{}).(*clockCell)
	return c
}

// Skew returns the wall-clock skew currently stored in the clock cell bound to
// ctx, or 0 if no cell is bound. Safe for concurrent use.
func Skew(ctx context.Context) time.Duration {
	if c := cellFrom(ctx); c != nil {
		return time.Duration(c.skew.Load())
	}
	return 0
}

// ResetClock stores zero skew into the clock cell bound to ctx, undoing any
// skew a Clock fault has set (a mid-test reset). A no-op if no cell is bound.
func ResetClock(ctx context.Context) {
	if c := cellFrom(ctx); c != nil {
		c.skew.Store(0)
	}
}

// Clock returns a fault that, when it fires, sets the skew of the clock cell
// bound to the context to d, so subsequent engine.Now(ctx) reads observe a wall
// clock shifted by d. It injects no error and no sleep (Apply returns nil), so
// it composes with other faults in the same rule. If no clock cell is bound to
// the context, Apply is a no-op.
func Clock(d time.Duration) Fault {
	return clockFault{d: d}
}

type clockFault struct {
	d time.Duration
}

func (c clockFault) Apply(ctx context.Context) error {
	if cell := cellFrom(ctx); cell != nil {
		cell.skew.Store(int64(c.d))
	}
	return nil
}

func (clockFault) Kind() Kind {
	return KindClock
}
