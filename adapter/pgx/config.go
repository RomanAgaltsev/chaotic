package pgx

import (
	"context"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	chaosnet "github.com/RomanAgaltsev/chaotic/adapter/net"
	"github.com/RomanAgaltsev/chaotic/engine"
)

// instrumentOptions holds the resolved toggles for InstrumentPoolConfig.
type instrumentOptions struct {
	dialFaults   bool
	queryLatency bool
}

// InstrumentOption configures InstrumentPoolConfig. Both mechanisms are enabled
// by default; use the WithoutX options to disable one.
type InstrumentOption func(*instrumentOptions)

// WithoutDialFaults disables connection-transport chaos (the chaosnet DialFunc
// wiring), leaving cfg.ConnConfig.DialFunc untouched.
func WithoutDialFaults() InstrumentOption {
	return func(o *instrumentOptions) { o.dialFaults = false }
}

// WithoutQueryLatency disables the query-latency QueryTracer, leaving
// cfg.ConnConfig.Tracer untouched.
func WithoutQueryLatency() InstrumentOption {
	return func(o *instrumentOptions) { o.queryLatency = false }
}

// InstrumentPoolConfig wires chaos into cfg at the pgx config level and returns
// the same *pgxpool.Config, so the pool built from it stays a genuine
// *pgxpool.Pool — adopters need no type change.
//
// By default it wires two independent interception points:
//
//   - Connection transport faults via chaosnet on cfg.ConnConfig.DialFunc
//     (latency, conn drop/reset, truncate). Any existing DialFunc is chained.
//   - Query latency via a pgx.QueryTracer on cfg.ConnConfig.Tracer. Any existing
//     tracer (e.g. otelpgx) is preserved and both are invoked.
//
// The tracer path injects LATENCY ONLY: a pgx.QueryTracer cannot change a
// query's result or error, so per-operation error matching is not available on
// the config path — use WrapPool for that. See the package doc's coverage matrix.
//
// A nil cfg or nil eng is a programmer error and panics, matching WrapPool.
func InstrumentPoolConfig(cfg *pgxpool.Config, eng *engine.Engine, opts ...InstrumentOption) *pgxpool.Config {
	if cfg == nil {
		panic("adapter/pgx: InstrumentPoolConfig requires a non-nil *pgxpool.Config")
	}
	if eng == nil {
		panic("adapter/pgx: InstrumentPoolConfig requires a non-nil *engine.Engine")
	}

	o := instrumentOptions{dialFaults: true, queryLatency: true}
	for _, opt := range opts {
		opt(&o)
	}

	if o.dialFaults {
		wireDialFunc(cfg, eng)
	}
	if o.queryLatency {
		wireQueryTracer(cfg, eng) // implemented in Task 3
	}
	return cfg
}

// wireDialFunc replaces cfg.ConnConfig.DialFunc with a chaosnet-wrapped dialer,
// chaining any dialer already set so an existing custom dialer still runs.
func wireDialFunc(cfg *pgxpool.Config, eng *engine.Engine) {
	var inner chaosnet.DialFunc
	if prev := cfg.ConnConfig.DialFunc; prev != nil {
		inner = chaosnet.DialFunc(prev)
	}
	cfg.ConnConfig.DialFunc = chaosnet.WrapDialer(eng, inner).DialContext
}

// wireQueryTracer sets cfg.ConnConfig.Tracer to a latency-injecting tracer. If a
// tracer is already set, both are invoked via a multiplexing tracer (the new one
// runs first so its latency precedes the existing tracer's bookkeeping).
func wireQueryTracer(cfg *pgxpool.Config, eng *engine.Engine) {
	chaos := &latencyTracer{eng: eng}
	if existing := cfg.ConnConfig.Tracer; existing != nil {
		cfg.ConnConfig.Tracer = multiTracer{chaos, existing}
		return
	}
	cfg.ConnConfig.Tracer = chaos
}

// traceActionKey carries the engine Action from TraceQueryStart to
// TraceQueryEnd so the held bound (e.g. a WithMaxConcurrent slot) is released.
type traceActionKey struct{}

// latencyTracer injects engine-configured latency at the query boundary.
//
// It can only apply latency: pgx's QueryTracer has no way to change a query's
// result or error, so any error a fired rule would inject is discarded here.
// A rule whose latency fault is ordered before its error fault still has its
// latency applied (engine applies faults in declared order). The call still
// consumes the rule's budget (Times, etc.).
type latencyTracer struct {
	eng *engine.Engine
}

func (t *latencyTracer) TraceQueryStart(ctx context.Context, _ *pgxv5.Conn, data pgxv5.TraceQueryStartData) context.Context {
	if !t.eng.Enabled() {
		return ctx
	}
	action := t.eng.Eval(ctx, opTrace(data.SQL, len(data.Args)))
	// Before applies latency (sleeping) and may return an error we cannot
	// surface through the tracer; discard it. The action is still carried to
	// TraceQueryEnd so After releases any held bound exactly once.
	_ = action.Before(ctx)
	return context.WithValue(ctx, traceActionKey{}, action)
}

func (t *latencyTracer) TraceQueryEnd(ctx context.Context, _ *pgxv5.Conn, data pgxv5.TraceQueryEndData) {
	if a, ok := ctx.Value(traceActionKey{}).(engine.Action); ok {
		finish(ctx, a, data.Err)
	}
}

// multiTracer fans a QueryTracer call out to several tracers in order. The
// context threaded through TraceQueryStart accumulates each tracer's additions.
type multiTracer []pgxv5.QueryTracer

func (m multiTracer) TraceQueryStart(ctx context.Context, conn *pgxv5.Conn, data pgxv5.TraceQueryStartData) context.Context {
	for _, t := range m {
		ctx = t.TraceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (m multiTracer) TraceQueryEnd(ctx context.Context, conn *pgxv5.Conn, data pgxv5.TraceQueryEndData) {
	for _, t := range m {
		t.TraceQueryEnd(ctx, conn, data)
	}
}

// Compile-time proof the tracer types satisfy pgx's QueryTracer.
var (
	_ pgxv5.QueryTracer = (*latencyTracer)(nil)
	_ pgxv5.QueryTracer = multiTracer(nil)
)
