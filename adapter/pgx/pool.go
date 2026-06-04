package pgx

import (
	"context"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ag4r/chaotic/engine"
)

// Pool wraps a *pgxpool.Pool and intercepts Query, QueryRow, Exec, SendBatch,
// Begin, BeginTx and Acquire. Other pool methods (Close, Stat, Config, Reset,
// Ping) pass through.
type Pool struct {
	b   poolBackend
	eng *engine.Engine
	raw *pgxpool.Pool // for Unwrap(); nil if constructed in tests with a fake.
}

// Query runs the engine's chaos for the operation, then delegates to the
// underlying pool's Query.
func (p *Pool) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if !p.eng.Enabled() {
		return p.b.Query(ctx, sql, args...)
	}
	op := opQuery("query", sql, len(args), false)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return nil, translate(err)
	}
	return p.b.Query(ctx, sql, args...)
}

// Exec runs the engine's chaos for the operation, then delegates to the
// underlying pool's Exec.
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if !p.eng.Enabled() {
		return p.b.Exec(ctx, sql, args...)
	}
	op := opQuery("exec", sql, len(args), false)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return pgconn.CommandTag{}, translate(err)
	}
	return p.b.Exec(ctx, sql, args...)
}

// QueryRow runs the engine's chaos for the operation, then delegates to the
// underlying pool's QueryRow. On chaos error it returns a row that yields the
// error from Scan.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if !p.eng.Enabled() {
		return p.b.QueryRow(ctx, sql, args...)
	}
	op := opQuery("queryrow", sql, len(args), false)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return chaosRow{err: translate(err)}
	}
	return p.b.QueryRow(ctx, sql, args...)
}

// SendBatch runs the engine's chaos for the operation, then delegates to the
// underlying pool's SendBatch. On chaos error it returns batch results that
// yield the error from every method.
func (p *Pool) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if !p.eng.Enabled() {
		return p.b.SendBatch(ctx, b)
	}
	op := opBatch(b.Len(), false)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return chaosBatch{err: translate(err)}
	}
	return p.b.SendBatch(ctx, b)
}

// Ping is a pass-through: chaos rules MUST NOT poison health checks.
func (p *Pool) Ping(ctx context.Context) error {
	return p.b.Ping(ctx)
}

// Close closes the underlying pool. Pass-through.
func (p *Pool) Close() {
	if p.raw != nil {
		p.raw.Close()
	}
}

// Stat returns the underlying pool's statistics. Pass-through.
func (p *Pool) Stat() *pgxpool.Stat {
	if p.raw == nil {
		return nil
	}
	return p.raw.Stat()
}

// Config returns the underlying pool's config. Pass-through.
func (p *Pool) Config() *pgxpool.Config {
	if p.raw == nil {
		return nil
	}
	return p.raw.Config()
}

// Reset closes all connections in the pool, forcing reacquisition. Pass-through.
func (p *Pool) Reset() {
	if p.raw != nil {
		p.raw.Reset()
	}
}

// Unwrap returns the underlying *pgxpool.Pool. Returns nil when the wrapper
// was constructed against a test fake (no concrete pool exists in that case).
func (p *Pool) Unwrap() *pgxpool.Pool { return p.raw }

// Begin runs the engine's chaos for the operation, then starts a transaction
// on the underlying pool and returns it wrapped in a *Tx.
func (p *Pool) Begin(ctx context.Context) (*Tx, error) {
	if !p.eng.Enabled() {
		inner, err := p.b.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &Tx{
			b:   inner,
			eng: p.eng,
		}, nil
	}
	op := opBegin("", "", false)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return nil, translate(err)
	}
	inner, err := p.b.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Tx{
		b:   inner,
		eng: p.eng,
	}, nil
}

// BeginTx runs the engine's chaos for the operation, then starts a transaction
// with the given options on the underlying pool and returns it wrapped in a *Tx.
func (p *Pool) BeginTx(ctx context.Context, txOpts pgxv5.TxOptions) (*Tx, error) {
	if !p.eng.Enabled() {
		inner, err := p.b.BeginTx(ctx, txOpts)
		if err != nil {
			return nil, err
		}
		return &Tx{
			b:   inner,
			eng: p.eng,
		}, nil
	}
	iso, access, deferrable := txOptsStrings(string(txOpts.IsoLevel), string(txOpts.AccessMode), txOpts.DeferrableMode == pgxv5.Deferrable)
	op := opBegin(iso, access, deferrable)
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return nil, translate(err)
	}
	inner, err := p.b.BeginTx(ctx, txOpts)
	if err != nil {
		return nil, err
	}
	return &Tx{
		b:   inner,
		eng: p.eng,
	}, nil
}

// Acquire runs the engine's chaos for the operation, then acquires a connection
// from the underlying pool and returns it wrapped in a *Conn.
func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
	if !p.eng.Enabled() {
		inner, err := p.b.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		return &Conn{
			b: &pooledConnBackend{
				c: inner,
			},
			eng: p.eng,
			raw: inner,
		}, nil
	}
	op := opAcquire()
	if err := p.eng.Eval(ctx, op).Before(ctx); err != nil {
		return nil, translate(err)
	}
	inner, err := p.b.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{
		b: &pooledConnBackend{
			c: inner,
		},
		eng: p.eng,
		raw: inner,
	}, nil
}
