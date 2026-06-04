package pgx

import (
	"context"
	"errors"

	"github.com/ag4r/chaotic/fault"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ag4r/chaotic/engine"
)

// Conn wraps either a *pgxpool.Conn (from Pool.Acquire) or a *pgx.Conn
// (from WrapConn). Intercepts Query, QueryRow, Exec, SendBatch, Begin, BeginTx.
// Ping is a pass-through. Release is a pass-through; it's a no-op on the
// standalone-conn variant.
type Conn struct {
	b   connBackend
	eng *engine.Engine
	raw any // *pgxpool.Conn or *pgx.Conn; populated by WrapConn / Pool.Acquire.
}

// Query runs the engine's chaos for the operation, then delegates to the
// underlying conn's Query.
func (c *Conn) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if !c.eng.Enabled() {
		return c.b.Query(ctx, sql, args...)
	}
	action, err := c.runChaos(ctx, opQuery("query", sql, len(args), false))
	if err != nil {
		finish(ctx, action, err)
		return nil, err
	}
	rows, qerr := c.b.Query(ctx, sql, args...)
	finish(ctx, action, qerr)
	return rows, qerr
}

// Exec runs the engine's chaos for the operation, then delegates to the
// underlying conn's Exec.
func (c *Conn) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if !c.eng.Enabled() {
		return c.b.Exec(ctx, sql, args...)
	}
	action, err := c.runChaos(ctx, opQuery("exec", sql, len(args), false))
	if err != nil {
		finish(ctx, action, err)
		return pgconn.CommandTag{}, err
	}
	tag, eerr := c.b.Exec(ctx, sql, args...)
	finish(ctx, action, eerr)
	return tag, eerr
}

// QueryRow runs the engine's chaos for the operation, then delegates to the
// underlying conn's QueryRow. On chaos error it returns a row that yields the
// error from Scan.
func (c *Conn) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if !c.eng.Enabled() {
		return c.b.QueryRow(ctx, sql, args...)
	}
	action, err := c.runChaos(ctx, opQuery("queryrow", sql, len(args), false))
	if err != nil {
		finish(ctx, action, err)
		return chaosRow{err: err}
	}
	row := c.b.QueryRow(ctx, sql, args...)
	finish(ctx, action, nil)
	return row
}

// SendBatch runs the engine's chaos for the operation, then delegates to the
// underlying conn's SendBatch. On chaos error it returns batch results that
// yield the error from every method.
func (c *Conn) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if !c.eng.Enabled() {
		return c.b.SendBatch(ctx, b)
	}
	action, err := c.runChaos(ctx, opBatch(b.Len(), false))
	if err != nil {
		finish(ctx, action, err)
		return chaosBatch{err: err}
	}
	results := c.b.SendBatch(ctx, b)
	finish(ctx, action, nil)
	return results
}

// Begin runs the engine's chaos for the operation, then starts a transaction
// on the underlying conn and returns it wrapped in a *Tx.
func (c *Conn) Begin(ctx context.Context) (*Tx, error) {
	if !c.eng.Enabled() {
		inner, err := c.b.Begin(ctx)
		if err != nil {
			return nil, err
		}
		return &Tx{b: inner, eng: c.eng}, nil
	}
	action, err := c.runChaos(ctx, opBegin("", "", false))
	if err != nil {
		finish(ctx, action, err)
		return nil, err
	}
	inner, ierr := c.b.Begin(ctx)
	finish(ctx, action, ierr)
	if ierr != nil {
		return nil, ierr
	}
	return &Tx{b: inner, eng: c.eng}, nil
}

// BeginTx runs the engine's chaos for the operation, then starts a transaction
// with the given options on the underlying conn and returns it wrapped in a *Tx.
func (c *Conn) BeginTx(ctx context.Context, txOpts pgxv5.TxOptions) (*Tx, error) {
	if !c.eng.Enabled() {
		inner, err := c.b.BeginTx(ctx, txOpts)
		if err != nil {
			return nil, err
		}
		return &Tx{b: inner, eng: c.eng}, nil
	}
	op := opBegin(string(txOpts.IsoLevel), string(txOpts.AccessMode), txOpts.DeferrableMode == pgxv5.Deferrable)
	action, err := c.runChaos(ctx, op)
	if err != nil {
		finish(ctx, action, err)
		return nil, err
	}
	inner, ierr := c.b.BeginTx(ctx, txOpts)
	finish(ctx, action, ierr)
	if ierr != nil {
		return nil, ierr
	}
	return &Tx{b: inner, eng: c.eng}, nil
}

// Ping is a pass-through. Chaos must not poison health checks.
func (c *Conn) Ping(ctx context.Context) error {
	return c.b.Ping(ctx)
}

// Release returns a pool-acquired conn to its pool. It's a no-op for the
// standalone-conn variant (WrapConn-origin), where there's no pool to return to.
func (c *Conn) Release() {
	if pc, ok := c.raw.(*pgxpool.Conn); ok && pc != nil {
		pc.Release()
	}
}

// Conn returns the underlying *pgx.Conn. For pool-acquired conns this is
// pgxpool.Conn.Conn(). For standalone conns this is the wrapped *pgx.Conn.
func (c *Conn) Conn() *pgxv5.Conn {
	switch v := c.raw.(type) {
	case *pgxpool.Conn:
		if v == nil {
			return nil
		}
		return v.Conn()
	case *pgxv5.Conn:
		return v
	default:
		return nil
	}
}

// Unwrap returns the wrapped value. Type is *pgxpool.Conn for pool-acquired
// conns, *pgx.Conn for WrapConn-origin conns, or nil if the wrapper was
// constructed against a test fake.
func (c *Conn) Unwrap() any { return c.raw }

// runChaos evaluates the engine and applies the ConnDrop poison if the
// resulting fault was ErrConnDrop. It returns the engine action (so the caller
// can report the outcome and release any held bound via finish) and the
// translated error (or nil on no chaos / chaos passed).
func (c *Conn) runChaos(ctx context.Context, op engine.Op) (engine.Action, error) {
	action := c.eng.Eval(ctx, op)
	err := action.Before(ctx)
	if err == nil {
		return action, nil
	}
	if errors.Is(err, fault.ErrConnDrop) {
		// Poison: close the underlying conn. We ignore Close's error — the
		// chaos error is what the caller needs to see.
		_ = c.b.Close(ctx)
	}
	return action, translate(err)
}
