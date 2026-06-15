// Package fault defines the fault primitives that rules execute when they
// fire. Each fault implements Apply(ctx); a non-nil error short-circuits the
// adapter's Action.Before chain and is delivered to the caller in the
// adapter's native error model.
package fault

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
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

// ErrDisconnect is the sentinel returned by Disconnect's Apply. Each adapter
// detects this sentinel and substitutes its native graceful-close error (an
// orderly FIN), as distinct from ErrConnDrop's hard reset.
var ErrDisconnect = errors.New("chaotic: graceful disconnect")

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

// Jittered sleeps for a uniformly random duration in [lo, hi]. Negative or
// zero values are treated as "no sleep". If hi <= lo, sleeps for lo.
func Jittered(lo, hi time.Duration) Fault {
	return jitteredFault{
		min: lo,
		max: hi,
	}
}

func (latencyFault) Kind() Kind {
	return KindLatency
}

// Duration reports the fixed sleep this fault injects, letting tooling (e.g. a
// RichObserver latency histogram) record it without running the fault.
func (l latencyFault) Duration() time.Duration {
	return l.d
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

func (jitteredFault) Kind() Kind {
	return KindJittered
}

// Duration reports the upper bound of the jitter window. The actual sleep is a
// random draw in [min, max] that Apply does not expose, so tooling that reads
// Duration sees the configured ceiling, not the per-call value.
func (j jitteredFault) Duration() time.Duration {
	return j.max
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

func (errorFault) Kind() Kind {
	return KindError
}

type panicFault struct {
	v any
}

func (p panicFault) Apply(_ context.Context) error {
	panic(p.v)
}

func (panicFault) Kind() Kind {
	return KindPanic
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

func (connDropFault) Kind() Kind {
	return KindConnDrop
}

// Disconnect returns ErrDisconnect, modeling an orderly connection close (a TCP
// FIN). Distinct from ConnDrop, which models a hard reset; real clients often
// take different code paths for the two.
func Disconnect() Fault { return disconnectFault{} }

type disconnectFault struct{}

func (disconnectFault) Apply(context.Context) error { return ErrDisconnect }

func (disconnectFault) Kind() Kind { return KindDisconnect }

// JitteredSeed is like Jittered but draws from a seeded PCG source, so the
// sequence of sleep durations is reproducible across runs with the same seed.
// Use it when a chaos test must be deterministically replayable. The draw is
// mutex-guarded. The fault is safe for concurrent use.
func JitteredSeed(lo, hi time.Duration, seed int64) Fault {
	return &seededJitter{
		min: lo,
		max: hi,
		rng: rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9E3779B97F4A7C15)), //nolint:gosec // non-cryptographic randomness is intentional
	}
}

type seededJitter struct {
	min, max time.Duration
	mu       sync.Mutex
	rng      *rand.Rand
}

func (j *seededJitter) draw() time.Duration {
	if j.max <= j.min {
		return j.min
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	span := int64(j.max - j.min)
	return j.min + time.Duration(j.rng.Int64N(span+1))
}

func (j *seededJitter) Apply(ctx context.Context) error {
	return sleep(ctx, j.draw())
}

func (*seededJitter) Kind() Kind {
	return KindJittered
}

// Duration reports the upper bound of the jitter window (see jitteredFault).
func (j *seededJitter) Duration() time.Duration {
	return j.max
}
