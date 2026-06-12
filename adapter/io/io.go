//go:build !chaos_off

package io

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// truncCap is the sticky byte cap a Truncate fault arms on a stream.
type truncCap struct {
	armed     bool
	remaining int
}

type reader struct {
	inner io.Reader
	eng   *engine.Engine
	cap   truncCap
}

type writer struct {
	inner io.Writer
	eng   *engine.Engine
	cap   truncCap
}

// WrapReader returns an io.Reader that consults eng on each Read.
func WrapReader(r io.Reader, eng *engine.Engine) io.Reader {
	return &reader{inner: r, eng: eng}
}

// WrapWriter returns an io.Writer that consults eng on each Write.
func WrapWriter(w io.Writer, eng *engine.Engine) io.Writer {
	return &writer{inner: w, eng: eng}
}

func opFor(method string) engine.Op {
	return engine.Op{Kind: engine.OpIO, Method: method}
}

func (r *reader) Read(b []byte) (int, error) {
	if !r.eng.Enabled() {
		return r.inner.Read(b)
	}
	if r.cap.armed {
		return r.cappedRead(b)
	}
	ctx := context.Background()
	action := r.eng.Eval(ctx, opFor("read"))
	err := action.Before(ctx)
	if err == nil {
		n, rerr := r.inner.Read(b)
		reportOutcome(ctx, action, rerr)
		return n, rerr
	}
	var sfe *fault.StreamFaultError
	if errors.As(err, &sfe) {
		reportOutcome(ctx, action, nil)
		return r.shapeRead(sfe, b)
	}
	reportOutcome(ctx, action, err)
	return 0, err
}

func (w *writer) Write(b []byte) (int, error) {
	if !w.eng.Enabled() {
		return w.inner.Write(b)
	}
	if w.cap.armed {
		return w.cappedWrite(b)
	}
	ctx := context.Background()
	action := w.eng.Eval(ctx, opFor("write"))
	err := action.Before(ctx)
	if err == nil {
		n, werr := w.inner.Write(b)
		reportOutcome(ctx, action, werr)
		return n, werr
	}
	var sfe *fault.StreamFaultError
	if errors.As(err, &sfe) {
		reportOutcome(ctx, action, nil)
		return w.shapeWrite(sfe, b)
	}
	reportOutcome(ctx, action, err)
	return 0, err
}

// shapeRead / shapeWrite / cappedRead / cappedWrite / slow are added in Tasks C3-C4.

// reportOutcome runs Outcome (if implemented) and After exactly once. Nil is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if action == nil {
		return
	}
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
	_ = action.After(ctx)
}

func (r *reader) shapeRead(sfe *fault.StreamFaultError, b []byte) (int, error) {
	switch sfe.Mode {
	case fault.StreamTruncate:
		r.cap.armed = true
		r.cap.remaining = sfe.Limit
		return r.cappedRead(b)
	case fault.StreamSlowRead:
		n, err := r.inner.Read(b)
		slow(sfe.Rate, n)
		return n, err
	default: // SlowWrite on a reader: mismatch, shape nothing
		return r.inner.Read(b)
	}
}

func (r *reader) cappedRead(b []byte) (int, error) { return r.inner.Read(b) }

func (w *writer) shapeWrite(sfe *fault.StreamFaultError, b []byte) (int, error) {
	switch sfe.Mode {
	case fault.StreamTruncate:
		w.cap.armed = true
		w.cap.remaining = sfe.Limit
		return w.cappedWrite(b)
	case fault.StreamSlowWrite:
		n, err := w.inner.Write(b)
		slow(sfe.Rate, n)
		return n, err
	default: // SlowRead on a writer: mismatch, shape nothing
		return w.inner.Write(b)
	}
}

func (w *writer) cappedWrite(b []byte) (int, error) { return w.inner.Write(b) }

// slow sleeps for the time it takes to move n bytes at rate bytes/sec. rate == 0
// blocks until the process exits (the "stream that never ends" fault); io has no
// context to cancel against.
func slow(rate, n int) {
	if rate == 0 {
		select {} // never returns
	}
	if n <= 0 {
		return
	}
	time.Sleep(time.Duration(float64(n) / float64(rate) * float64(time.Second)))
}

var _ = time.Second // referenced once slow() lands in Task C3
