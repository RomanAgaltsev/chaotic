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
package pgx
