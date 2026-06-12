package pgx

import (
	"context"
	"errors"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// runChaos evaluates the engine, applies the ConnDrop poison via the
// underlying conn (when the Tx's Conn() returns non-nil), and returns the
// engine action (so the caller can report the outcome and release any held
// bound via finish) and the translated error.
func (t *Tx) runChaos(ctx context.Context, op engine.Op) (engine.Action, error) {
	action := t.eng.Eval(ctx, op)
	err := action.Before(ctx)
	if err == nil {
		return action, nil
	}
	if errors.Is(err, fault.ErrConnDrop) {
		if c := t.b.Conn(); c != nil {
			_ = c.Close(ctx)
		}
	}
	return action, translate(err)
}

// Tx wraps a pgx.Tx and intercepts Query, QueryRow, Exec, SendBatch.
// Commit, Rollback, and other Tx methods pass through unchanged (per spec §5.2).
// Satisfies pgxv5.Tx so it drops into any code that takes pgx.Tx.
type Tx struct {
	b   pgxv5.Tx
	eng *engine.Engine
}

// Query runs the engine's chaos for the operation, then delegates to the
// underlying tx's Query.
func (t *Tx) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if !t.eng.Enabled() {
		return t.b.Query(ctx, sql, args...)
	}
	action, err := t.runChaos(ctx, opQuery("query", sql, len(args), true))
	if err != nil {
		finish(ctx, action, err)
		return nil, err
	}
	rows, qerr := t.b.Query(ctx, sql, args...)
	finish(ctx, action, qerr)
	return rows, qerr
}

// Exec runs the engine's chaos for the operation, then delegates to the
// underlying tx's Exec.
func (t *Tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if !t.eng.Enabled() {
		return t.b.Exec(ctx, sql, args...)
	}
	action, err := t.runChaos(ctx, opQuery("exec", sql, len(args), true))
	if err != nil {
		finish(ctx, action, err)
		return pgconn.CommandTag{}, err
	}
	tag, eerr := t.b.Exec(ctx, sql, args...)
	finish(ctx, action, eerr)
	return tag, eerr
}

// QueryRow runs the engine's chaos for the operation, then delegates to the
// underlying tx's QueryRow. On chaos error it returns a row that yields the
// error from Scan.
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if !t.eng.Enabled() {
		return t.b.QueryRow(ctx, sql, args...)
	}
	action, err := t.runChaos(ctx, opQuery("queryrow", sql, len(args), true))
	if err != nil {
		finish(ctx, action, err)
		return chaosRow{err: err}
	}
	row := t.b.QueryRow(ctx, sql, args...)
	finish(ctx, action, nil)
	return row
}

// SendBatch runs the engine's chaos for the operation, then delegates to the
// underlying tx's SendBatch. On chaos error it returns batch results that
// yield the error from every method.
func (t *Tx) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if !t.eng.Enabled() {
		return t.b.SendBatch(ctx, b)
	}
	action, err := t.runChaos(ctx, opBatch(b.Len(), true))
	if err != nil {
		finish(ctx, action, err)
		return chaosBatch{err: err}
	}
	results := t.b.SendBatch(ctx, b)
	finish(ctx, action, nil)
	return results
}

// Begin starts a pseudo-nested transaction (savepoint). Pass-through.
func (t *Tx) Begin(ctx context.Context) (pgxv5.Tx, error) {
	return t.b.Begin(ctx)
}

// Commit commits the transaction. Pass-through.
func (t *Tx) Commit(ctx context.Context) error {
	return t.b.Commit(ctx)
}

// Rollback rolls the transaction back. Pass-through.
func (t *Tx) Rollback(ctx context.Context) error {
	return t.b.Rollback(ctx)
}

// CopyFrom performs a bulk copy into the given table. Pass-through.
func (t *Tx) CopyFrom(ctx context.Context, tableName pgxv5.Identifier, columnNames []string, rowSrc pgxv5.CopyFromSource) (int64, error) {
	return t.b.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// LargeObjects returns the large objects API for the transaction. Pass-through.
func (t *Tx) LargeObjects() pgxv5.LargeObjects {
	return t.b.LargeObjects()
}

// Prepare creates a prepared statement on the transaction. Pass-through.
func (t *Tx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return t.b.Prepare(ctx, name, sql)
}

// Conn returns the underlying *pgx.Conn for the transaction. Pass-through.
func (t *Tx) Conn() *pgxv5.Conn {
	return t.b.Conn()
}

// Compile-time assertion that *Tx satisfies the pgx.Tx interface.
var _ pgxv5.Tx = (*Tx)(nil)
