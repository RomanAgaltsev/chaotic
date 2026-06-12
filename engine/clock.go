package engine

import (
	"context"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// Now returns the current wall-clock time skewed by any fault.Clock active on
// ctx: time.Now().Add(fault.Skew(ctx)). With no skew (no clock cell bound, no
// Clock fault fired, or a chaos_off build) it is effectively time.Now(). Use
// Now instead of time.Now() in code whose clock-dependent behavior you want
// chaos to exercise (deadline math, expiry/token windows, timezone logic).
//
// The skew shifts both the wall and monotonic readings of the returned time, so
// elapsed-time subtraction between two Now reads is unaffected (a wrong clock
// must not warp how much time elapsed), while wall-clock field reads (Hour,
// Year, In) and comparisons against fixed external timestamps reflect the skew.
// Subtracting a skewed Now from a separately captured time.Now() cancels the
// skew via the shared monotonic reading.
func Now(ctx context.Context) time.Time {
	return time.Now().Add(fault.Skew(ctx))
}

// Since is the skew-aware equivalent of time.Since: engine.Now(ctx).Sub(t).
func Since(ctx context.Context, t time.Time) time.Duration {
	return Now(ctx).Sub(t)
}

// Until is the skew-aware equivalent of time.Until: t.Sub(engine.Now(ctx)).
func Until(ctx context.Context, t time.Time) time.Duration {
	return t.Sub(Now(ctx))
}
