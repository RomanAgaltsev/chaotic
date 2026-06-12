package pgx

import (
	"context"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RomanAgaltsev/chaotic/engine"
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
	action := p.eng.Eval(ctx, opQuery("query", sql, len(args), false))
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return nil, translate(err)
	}
	rows, err := p.b.Query(ctx, sql, args...)
	finish(ctx, action, err)
	return rows, err
}

// Exec runs the engine's chaos for the operation, then delegates to the
// underlying pool's Exec.
func (p *Pool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if !p.eng.Enabled() {
		return p.b.Exec(ctx, sql, args...)
	}
	action := p.eng.Eval(ctx, opQuery("exec", sql, len(args), false))
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return pgconn.CommandTag{}, translate(err)
	}
	tag, err := p.b.Exec(ctx, sql, args...)
	finish(ctx, action, err)
	return tag, err
}

// QueryRow runs the engine's chaos for the operation, then delegates to the
// underlying pool's QueryRow. On chaos error it returns a row that yields the
// error from Scan.
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if !p.eng.Enabled() {
		return p.b.QueryRow(ctx, sql, args...)
	}
	action := p.eng.Eval(ctx, opQuery("queryrow", sql, len(args), false))
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return chaosRow{err: translate(err)}
	}
	// The real outcome surfaces only from Scan, which we do not see; report a
	// nil outcome and release the bound now so the slot is not held for a Scan
	// that may never come.
	row := p.b.QueryRow(ctx, sql, args...)
	finish(ctx, action, nil)
	return row
}

// SendBatch runs the engine's chaos for the operation, then delegates to the
// underlying pool's SendBatch. On chaos error it returns batch results that
// yield the error from every method.
func (p *Pool) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if !p.eng.Enabled() {
		return p.b.SendBatch(ctx, b)
	}
	action := p.eng.Eval(ctx, opBatch(b.Len(), false))
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return chaosBatch{err: translate(err)}
	}
	results := p.b.SendBatch(ctx, b)
	finish(ctx, action, nil)
	return results
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
		return &Tx{b: inner, eng: p.eng}, nil
	}
	action := p.eng.Eval(ctx, opBegin("", "", false))
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return nil, translate(err)
	}
	inner, err := p.b.Begin(ctx)
	finish(ctx, action, err)
	if err != nil {
		return nil, err
	}
	return &Tx{b: inner, eng: p.eng}, nil
}

// BeginTx runs the engine's chaos for the operation, then starts a transaction
// with the given options on the underlying pool and returns it wrapped in a *Tx.
func (p *Pool) BeginTx(ctx context.Context, txOpts pgxv5.TxOptions) (*Tx, error) {
	if !p.eng.Enabled() {
		inner, err := p.b.BeginTx(ctx, txOpts)
		if err != nil {
			return nil, err
		}
		return &Tx{b: inner, eng: p.eng}, nil
	}
	op := opBegin(string(txOpts.IsoLevel), string(txOpts.AccessMode), txOpts.DeferrableMode == pgxv5.Deferrable)
	action := p.eng.Eval(ctx, op)
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return nil, translate(err)
	}
	inner, err := p.b.BeginTx(ctx, txOpts)
	finish(ctx, action, err)
	if err != nil {
		return nil, err
	}
	return &Tx{b: inner, eng: p.eng}, nil
}

// Acquire runs the engine's chaos for the operation, then acquires a connection
// from the underlying pool and returns it wrapped in a *Conn.
func (p *Pool) Acquire(ctx context.Context) (*Conn, error) {
	if !p.eng.Enabled() {
		inner, err := p.b.Acquire(ctx)
		if err != nil {
			return nil, err
		}
		return &Conn{b: &pooledConnBackend{c: inner}, eng: p.eng, raw: inner}, nil
	}
	action := p.eng.Eval(ctx, opAcquire())
	if err := action.Before(ctx); err != nil {
		finish(ctx, action, err)
		return nil, translate(err)
	}
	inner, err := p.b.Acquire(ctx)
	finish(ctx, action, err)
	if err != nil {
		return nil, err
	}
	return &Conn{b: &pooledConnBackend{c: inner}, eng: p.eng, raw: inner}, nil
}
