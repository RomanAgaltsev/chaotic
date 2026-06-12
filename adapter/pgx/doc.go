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
package pgx
