// Package fault defines the fault primitives that rules execute when they
// fire. Each fault implements Apply(ctx); a non-nil error short-circuits the
// adapter's Action.Before chain and is delivered to the caller in the
// adapter's native error model.
package fault

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// Fault is one chaos primitive. Apply may sleep, return an error, or panic.
// A return of nil means the fault completed without affecting the call.
type Fault interface {
	Apply(ctx context.Context) error
}

// ErrConnDrop is the sentinel returned by ConnDrop's Apply. Each adapter
// detects this sentinel via errors.Is and substitutes its native
// connection-drop error (driver.ErrBadConn, status.Unavailable, etc.).
var ErrConnDrop = errors.New("chaotic: connection drop")

// Latency sleeps for d. Returns ctx.Err() if the context is canceled first.
// A non-positive d returns immediately.
func Latency(d time.Duration) Fault {
	return latencyFault{
		d: d,
	}
}

type latencyFault struct {
	d time.Duration
}

func (l latencyFault) Apply(ctx context.Context) error {
	return sleep(ctx, l.d)
}

// Jittered sleeps for a uniformly random duration in [min, max]. Negative or
// zero values are treated as "no sleep". If max <= min, sleeps for min.
func Jittered(min, max time.Duration) Fault {
	return jitteredFault{
		min: min,
		max: max,
	}
}

type jitteredFault struct {
	min, max time.Duration
}

func (j jitteredFault) Apply(ctx context.Context) error {
	d := j.min
	if j.max > j.min {
		span := int64(j.max - j.min)
		d = j.min + time.Duration(rand.Int64N(span+1)) //nolint:gosec // non-cryptographic randomness is intentional for jitter duration
	}
	return sleep(ctx, d)
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Error returns err verbatim from Apply. The adapter is responsible for
// wrapping it into its native error model (e.g., &url.Error{Op: "chaos"} for
// http). A nil err makes Apply a no-op.
func Error(err error) Fault {
	return errorFault{
		err: err,
	}
}

type errorFault struct {
	err error
}

func (e errorFault) Apply(ctx context.Context) error {
	return e.err
}

// Panic calls panic(v) from Apply. The panic propagates through the action,
// through the adapter, out to the caller. Recovery is the caller's
// responsibility.
func Panic(v any) Fault {
	return panicFault{
		v: v,
	}
}

type panicFault struct {
	v any
}

func (p panicFault) Apply(ctx context.Context) error {
	panic(p.v)
}

// ConnDrop returns ErrConnDrop. Each adapter detects this sentinel and
// substitutes its native connection-drop error.
func ConnDrop() Fault {
	return connDropFault{}
}

type connDropFault struct{}

func (connDropFault) Apply(context.Context) error {
	return ErrConnDrop
}
