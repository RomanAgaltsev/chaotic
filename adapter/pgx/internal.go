package pgx

import (
	"context"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// poolBackend is what *Pool delegates to. *pgxpool.Pool satisfies this
// structurally. Tests substitute a fake.
type poolBackend interface {
	Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	Begin(ctx context.Context) (pgxv5.Tx, error)
	BeginTx(ctx context.Context, txOpts pgxv5.TxOptions) (pgxv5.Tx, error)
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	Ping(ctx context.Context) error
}

// connBackend is what *Conn delegates to.
//
// *pgx.Conn satisfies this directly. *pgxpool.Conn does NOT satisfy it
// (its termination API is Release() + Conn().Close(ctx) rather than a
// single Close(ctx)); pooledConnBackend below adapts it.
type connBackend interface {
	Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	Begin(ctx context.Context) (pgxv5.Tx, error)
	BeginTx(ctx context.Context, txOpts pgxv5.TxOptions) (pgxv5.Tx, error)
	Ping(ctx context.Context) error
	// Close terminates the underlying conn. For ConnDrop poisoning.
	Close(ctx context.Context) error
}

// pooledConnBackend adapts *pgxpool.Conn to connBackend. It implements
// Close(ctx) by closing the underlying *pgx.Conn (which forces pgxpool
// to evict the pooled entry on next Release).
type pooledConnBackend struct {
	c *pgxpool.Conn
}

func (p *pooledConnBackend) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	return p.c.Query(ctx, sql, args...)
}

func (p *pooledConnBackend) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	return p.c.QueryRow(ctx, sql, args...)
}

func (p *pooledConnBackend) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.c.Exec(ctx, sql, args...)
}

func (p *pooledConnBackend) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	return p.c.SendBatch(ctx, b)
}

func (p *pooledConnBackend) Begin(ctx context.Context) (pgxv5.Tx, error) {
	return p.c.Begin(ctx)
}

func (p *pooledConnBackend) BeginTx(ctx context.Context, o pgxv5.TxOptions) (pgxv5.Tx, error) {
	return p.c.BeginTx(ctx, o)
}

func (p *pooledConnBackend) Ping(ctx context.Context) error {
	return p.c.Ping(ctx)
}

func (p *pooledConnBackend) Close(ctx context.Context) error {
	return p.c.Conn().Close(ctx)
}

// standaloneConnBackend wraps *pgx.Conn. It already satisfies connBackend
// because *pgx.Conn.Close has the right signature; this wrapper exists only
// to keep a typed reference for Unwrap() and to make Release() a no-op.
type standaloneConnBackend struct {
	*pgxv5.Conn
}
