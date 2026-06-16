package pgx

import (
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

// wireQueryTracer is implemented in Task 3.
func wireQueryTracer(cfg *pgxpool.Config, eng *engine.Engine) {}
