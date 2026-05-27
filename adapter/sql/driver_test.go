package sql_test

import (
	"context"
	"database/sql"
	dbdrv "database/sql/driver"
	"errors"
	"io"
	"testing"
	"time"

	chaossql "github.com/ag4r/chaotic/adapter/sql"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// --- in-memory driver shim for testing ---

type fakeDriver struct{}

func (fakeDriver) Open(name string) (dbdrv.Conn, error) {
	return &fakeConn{}, nil
}

type fakeConn struct {
	closed bool
}

func (c *fakeConn) Prepare(query string) (dbdrv.Stmt, error) {
	return &fakeStmt{q: query}, nil
}

func (c *fakeConn) Close() error {
	c.closed = true
	return nil
}

func (c *fakeConn) Begin() (dbdrv.Tx, error) {
	return &fakeTx{}, nil
}

func (c *fakeConn) Ping(ctx context.Context) error {
	return nil
}

func (c *fakeConn) ExecContext(ctx context.Context, q string, args []dbdrv.NamedValue) (dbdrv.Result, error) {
	return fakeResult{}, nil
}

func (c *fakeConn) QueryContext(ctx context.Context, q string, args []dbdrv.NamedValue) (dbdrv.Rows, error) {
	return &fakeRows{}, nil
}

func (c *fakeConn) BeginTx(ctx context.Context, _ dbdrv.TxOptions) (dbdrv.Tx, error) {
	return &fakeTx{}, nil
}

type fakeStmt struct {
	q string
}

func (s *fakeStmt) Close() error {
	return nil
}

func (s *fakeStmt) NumInput() int {
	return -1
}

func (s *fakeStmt) Exec(args []dbdrv.Value) (dbdrv.Result, error) {
	return &fakeResult{}, nil
}

func (s *fakeStmt) Query(args []dbdrv.Value) (dbdrv.Rows, error) {
	return &fakeRows{}, nil
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (fakeResult) RowsAffected() (int64, error) {
	return 0, nil
}

type fakeRows struct {
	done bool
}

func (r *fakeRows) Columns() []string {
	return []string{"x"}
}

func (r *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Next(dest []dbdrv.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = int64(1)
	return nil
}

type fakeTx struct{}

func (t *fakeTx) Commit() error {
	return nil
}

func (t *fakeTx) Rollback() error {
	return nil
}

func init() {
	sql.Register("chaosfake", &fakeDriver{})
}

func registerChaos(t *testing.T, name string, e *engine.Engine) {
	t.Helper()
	// chaossql.Register may be called only once per name, use unique names.
	chaossql.Register(name, "chaosfake", e)
}

// --- tests ---

func TestSqlNoOpWhenEngineEmpty(t *testing.T) {
	registerChaos(t, "chaos:noop", engine.New())
	db, err := sql.Open("chaos:noop", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.ExecContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
}

func TestSqlErrorFaultReturnsError(t *testing.T) {
	target := errors.New("boom")
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.WithFault(fault.Error(target)),
	))
	registerChaos(t, "chaos:err", e)
	db, _ := sql.Open("chaos:err", "")
	defer func() { _ = db.Close() }()
	_, err := db.ExecContext(context.Background(), "SELECT 1")
	if !errors.Is(err, target) {
		t.Fatalf("err = %v, want errors.Is(target) == true", err)
	}
}

func TestSqlConnDropFaultReturnsBadConn(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.WithFault(fault.ConnDrop()),
	))
	registerChaos(t, "chaos:drop", e)
	db, _ := sql.Open("chaos:drop", "")
	defer func() { _ = db.Close() }()
	db.SetMaxIdleConns(0)
	// database/sql retries on ErrBadConn. With only fake drivers, we'll see
	// the error after retries. Get a single connection and exec directly.
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_, err = conn.ExecContext(context.Background(), "SELECT 1")
	if !errors.Is(err, dbdrv.ErrBadConn) {
		t.Fatalf("err = %v, want driver.ErrBadConn", err)
	}
}

func TestSqlLatencyFaultDelays(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.WithFault(fault.Latency(40*time.Millisecond)),
	))
	registerChaos(t, "chaos:lat", e)
	db, _ := sql.Open("chaos:lat", "")
	defer func() { _ = db.Close() }()
	start := time.Now()
	if _, err := db.ExecContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatal("Exec returned too quickly")
	}
}

func TestSqlOpNameIsClassified(t *testing.T) {
	var gotName string
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchPredicate(func(_ context.Context, op engine.Op) bool {
			gotName = op.Name
			return false
		}),
	))
	registerChaos(t, "chaos:classify", e)
	db, _ := sql.Open("chaos:classify", "")
	defer func() { _ = db.Close() }()
	_, _ = db.ExecContext(context.Background(), "INSERT INTO users VALUES (1)")
	if gotName != "INSERT" {
		t.Fatalf("Op.Name = %q, want INSERT", gotName)
	}
}
