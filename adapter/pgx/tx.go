package pgx

import (
	"context"
	"errors"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// runChaos evaluates the engine, applies the ConnDrop poison via the
// underlying conn (when the Tx's Conn() returns non-nil), and returns the
// translated error.
func (t *Tx) runChaos(ctx context.Context, op engine.Op) error {
	err := t.eng.Eval(ctx, op).Before(ctx)
	if err == nil {
		return nil
	}
	if errors.Is(err, fault.ErrConnDrop) {
		if c := t.b.Conn(); c != nil {
			_ = c.Close(ctx)
		}
	}
	return translate(err)
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
	if err := t.runChaos(ctx, opQuery("query", sql, len(args), true)); err != nil {
		return nil, err
	}
	return t.b.Query(ctx, sql, args...)
}

// Exec runs the engine's chaos for the operation, then delegates to the
// underlying tx's Exec.
func (t *Tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if !t.eng.Enabled() {
		return t.b.Exec(ctx, sql, args...)
	}
	if err := t.runChaos(ctx, opQuery("exec", sql, len(args), true)); err != nil {
		return pgconn.CommandTag{}, err
	}
	return t.b.Exec(ctx, sql, args...)
}

// QueryRow runs the engine's chaos for the operation, then delegates to the
// underlying tx's QueryRow. On chaos error it returns a row that yields the
// error from Scan.
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if !t.eng.Enabled() {
		return t.b.QueryRow(ctx, sql, args...)
	}
	if err := t.runChaos(ctx, opQuery("queryrow", sql, len(args), true)); err != nil {
		return chaosRow{err: err}
	}
	return t.b.QueryRow(ctx, sql, args...)
}

// SendBatch runs the engine's chaos for the operation, then delegates to the
// underlying tx's SendBatch. On chaos error it returns batch results that
// yield the error from every method.
func (t *Tx) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if !t.eng.Enabled() {
		return t.b.SendBatch(ctx, b)
	}
	if err := t.runChaos(ctx, opBatch(b.Len(), true)); err != nil {
		return chaosBatch{err: err}
	}
	return t.b.SendBatch(ctx, b)
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
