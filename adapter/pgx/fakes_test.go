package pgx

import (
	"context"
	"fmt"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stubMissing panics with a clear message when a fake method is invoked
// without being stubbed. Tests should set the relevant onX field if they
// expect the backend to be called.
func stubMissing(method string) {
	panic(fmt.Sprintf("pgx adapter test fake: %s not stubbed", method))
}

type fakePool struct {
	onQuery     func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	onQueryRow  func(ctx context.Context, sql string, args ...any) pgxv5.Row
	onExec      func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	onSendBatch func(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	onBegin     func(ctx context.Context) (pgxv5.Tx, error)
	onBeginTx   func(ctx context.Context, o pgxv5.TxOptions) (pgxv5.Tx, error)
	onAcquire   func(ctx context.Context) (*pgxpool.Conn, error)
	onPing      func(ctx context.Context) error
}

func (f *fakePool) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if f.onQuery == nil {
		stubMissing("Query")
	}
	return f.onQuery(ctx, sql, args...)
}

func (f *fakePool) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if f.onQueryRow == nil {
		stubMissing("QueryRow")
	}
	return f.onQueryRow(ctx, sql, args...)
}

func (f *fakePool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.onExec == nil {
		stubMissing("Exec")
	}
	return f.onExec(ctx, sql, args...)
}

func (f *fakePool) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if f.onSendBatch == nil {
		stubMissing("SendBatch")
	}
	return f.onSendBatch(ctx, b)
}

func (f *fakePool) Begin(ctx context.Context) (pgxv5.Tx, error) {
	if f.onBegin == nil {
		stubMissing("Begin")
	}
	return f.onBegin(ctx)
}

func (f *fakePool) BeginTx(ctx context.Context, o pgxv5.TxOptions) (pgxv5.Tx, error) {
	if f.onBeginTx == nil {
		stubMissing("BeginTx")
	}
	return f.onBeginTx(ctx, o)
}

func (f *fakePool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	if f.onAcquire == nil {
		stubMissing("Acquire")
	}
	return f.onAcquire(ctx)
}

func (f *fakePool) Ping(ctx context.Context) error {
	if f.onPing == nil {
		stubMissing("Ping")
	}
	return f.onPing(ctx)
}

type fakeConn struct {
	onQuery     func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	onQueryRow  func(ctx context.Context, sql string, args ...any) pgxv5.Row
	onExec      func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	onSendBatch func(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	onBegin     func(ctx context.Context) (pgxv5.Tx, error)
	onBeginTx   func(ctx context.Context, o pgxv5.TxOptions) (pgxv5.Tx, error)
	onPing      func(ctx context.Context) error
	onClose     func(ctx context.Context) error
}

func (f *fakeConn) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if f.onQuery == nil {
		stubMissing("Query")
	}
	return f.onQuery(ctx, sql, args...)
}

func (f *fakeConn) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if f.onQueryRow == nil {
		stubMissing("QueryRow")
	}
	return f.onQueryRow(ctx, sql, args...)
}

func (f *fakeConn) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.onExec == nil {
		stubMissing("Exec")
	}
	return f.onExec(ctx, sql, args...)
}

func (f *fakeConn) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if f.onSendBatch == nil {
		stubMissing("SendBatch")
	}
	return f.onSendBatch(ctx, b)
}

func (f *fakeConn) Begin(ctx context.Context) (pgxv5.Tx, error) {
	if f.onBegin == nil {
		stubMissing("Begin")
	}
	return f.onBegin(ctx)
}

func (f *fakeConn) BeginTx(ctx context.Context, o pgxv5.TxOptions) (pgxv5.Tx, error) {
	if f.onBeginTx == nil {
		stubMissing("BeginTx")
	}
	return f.onBeginTx(ctx, o)
}

func (f *fakeConn) Ping(ctx context.Context) error {
	if f.onPing == nil {
		stubMissing("Ping")
	}
	return f.onPing(ctx)
}

func (f *fakeConn) Close(ctx context.Context) error {
	if f.onClose == nil {
		return nil // Close is allowed to be unstubbed; tests using ConnDrop poison set it.
	}
	return f.onClose(ctx)
}

// fakeTx satisfies pgxv5.Tx. It's the value returned by fakePool.Begin /
// BeginTx and fakeConn.Begin / BeginTx in tests.
type fakeTx struct {
	onQuery     func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error)
	onQueryRow  func(ctx context.Context, sql string, args ...any) pgxv5.Row
	onExec      func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	onSendBatch func(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults
	onCommit    func(ctx context.Context) error
	onRollback  func(ctx context.Context) error
}

func (f *fakeTx) Begin(ctx context.Context) (pgxv5.Tx, error) {
	return nil, fmt.Errorf("fakeTx.Begin not stubbed")
}

func (f *fakeTx) Commit(ctx context.Context) error {
	if f.onCommit == nil {
		return nil
	}
	return f.onCommit(ctx)
}

func (f *fakeTx) Rollback(ctx context.Context) error {
	if f.onRollback == nil {
		return nil
	}
	return f.onRollback(ctx)
}

func (f *fakeTx) CopyFrom(ctx context.Context, tableName pgxv5.Identifier, columnNames []string, rowSrc pgxv5.CopyFromSource) (int64, error) {
	return 0, fmt.Errorf("fakeTx.CopyFrom not stubbed")
}

func (f *fakeTx) SendBatch(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
	if f.onSendBatch == nil {
		stubMissing("fakeTx.SendBatch")
	}
	return f.onSendBatch(ctx, b)
}

func (f *fakeTx) LargeObjects() pgxv5.LargeObjects {
	panic("fakeTx.LargeObjects not stubbed")
}

func (f *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, fmt.Errorf("fakeTx.Prepare not stubbed")
}

func (f *fakeTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.onExec == nil {
		stubMissing("fakeTx.Exec")
	}
	return f.onExec(ctx, sql, args...)
}

func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
	if f.onQuery == nil {
		stubMissing("fakeTx.Query")
	}
	return f.onQuery(ctx, sql, args...)
}

func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgxv5.Row {
	if f.onQueryRow == nil {
		stubMissing("fakeTx.QueryRow")
	}
	return f.onQueryRow(ctx, sql, args...)
}
func (f *fakeTx) Conn() *pgxv5.Conn { return nil }

// Compile-time assertions:
var (
	_ poolBackend = (*fakePool)(nil)
	_ connBackend = (*fakeConn)(nil)
	_ pgxv5.Tx    = (*fakeTx)(nil)
)
