package pgx

import (
	"context"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the chaos-injectable surface shared by *pgxpool.Pool and *Pool.
//
// It lists exactly the pool methods whose signatures are identical on both
// types, so a consumer can declare a field or parameter of type Querier and
// inject the real *pgxpool.Pool in production and a chaos-wrapped *Pool (from
// WrapPool) in tests — with no other code change.
//
// Begin, BeginTx and Acquire are intentionally NOT part of Querier: *Pool wraps
// transactions and acquired connections in its own *Tx / *Conn types, whose
// return types differ from *pgxpool.Pool's (pgx.Tx and *pgxpool.Conn). No single
// Go interface can be satisfied by both for those methods. Code that needs the
// transaction or Acquire path must hold the concrete *Pool (or *pgxpool.Pool)
// type. This is a deliberate, permanent v1 design choice — see the package doc.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	Ping(ctx context.Context) error
}

// Compile-time proof that both the chaos wrapper and the real pgxpool pool
// satisfy Querier. If a future pgx release changes one of these signatures,
// this line breaks the build instead of silently breaking adopters.
var (
	_ Querier = (*Pool)(nil)
	_ Querier = (*pgxpool.Pool)(nil)
)
