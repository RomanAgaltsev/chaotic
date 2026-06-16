// Package pgx is a chaos adapter for github.com/jackc/pgx/v5 — native pgx
// (not the database/sql bridge). Wrap a *pgxpool.Pool with WrapPool, or a
// *pgx.Conn with WrapConn, and the returned *Pool / *Conn intercept Query,
// QueryRow, Exec, SendBatch, Begin, BeginTx, and (for the pool) Acquire,
// consulting the chaotic engine on each call.
//
// Users typically alias this package to avoid colliding with the upstream
// pgx package:
//
//	import chaospgx "github.com/RomanAgaltsev/chaotic/adapter/pgx"
//
// # Injecting behind an interface
//
// To adopt chaos without a consumer type change, type your field or parameter
// against the exported Querier interface instead of *pgxpool.Pool:
//
//	type Store struct{ DB chaospgx.Querier }
//
// Both *pgxpool.Pool and *chaospgx.Pool (from WrapPool) satisfy Querier, so the
// real pool is injected in production and the chaos wrapper in tests.
//
// Querier covers Query, QueryRow, Exec, SendBatch and Ping — the methods whose
// signatures match on both types. Begin, BeginTx and Acquire are excluded by
// design: the wrapper returns its own *Tx / *Conn types, whose signatures
// diverge from the real pool's, so no single interface can satisfy both. This
// is a permanent v1 choice: chaotic does not introduce a transaction-carrying
// shared interface (which would require a breaking return-type change). Code on
// the transaction or Acquire path holds the concrete *chaospgx.Pool type.
//
// # Config-level instrumentation (zero type change)
//
// InstrumentPoolConfig wires chaos into a *pgxpool.Config and returns it, so the
// pool stays a genuine *pgxpool.Pool — ideal for test-only injection (build the
// instrumented config in the test) and gated production use:
//
//	cfg, _ := pgxpool.ParseConfig(dsn)
//	cfg = chaospgx.InstrumentPoolConfig(cfg, eng)
//	pool, _ := pgxpool.NewWithConfig(ctx, cfg) // still *pgxpool.Pool
//
// It wires two independent interception points (each toggleable via options):
// transport faults on ConnConfig.DialFunc (via chaosnet) and query latency on
// ConnConfig.Tracer (chaining any existing tracer such as otelpgx).
//
// Fault coverage compared to WrapPool:
//
//	Path                          latency  conn drop/reset  per-op error match
//	InstrumentPoolConfig          yes      yes (DialFunc)   no
//	WrapPool                      yes      yes              yes
//
// The config path's tracer cannot inject errors into query results — a
// pgx.QueryTracer can only add latency. For per-operation error matching, use
// WrapPool. Enabling both the DialFunc and tracer mechanisms creates two
// independent interception points; a single query may then be evaluated by the
// engine more than once, which can double-consume budget-limited rules (Times,
// failure budgets). For budget-sensitive rules, enable only one mechanism.
package pgx
