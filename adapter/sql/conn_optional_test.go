//go:build !chaos_off

package sql_test

import (
	"context"
	"database/sql"
	dbdrv "database/sql/driver"
	"errors"
	"testing"

	chaossql "github.com/RomanAgaltsev/chaotic/adapter/sql"
	"github.com/RomanAgaltsev/chaotic/engine"
)

// --- a wrapped driver implementing every optional interface ---

type fullDriver struct{}

func (fullDriver) Open(string) (dbdrv.Conn, error) { return &fullConn{}, nil }

type fullConn struct {
	resetCalled   bool
	checkCalled   bool
	prepareCtxHit bool
}

func (c *fullConn) Prepare(string) (dbdrv.Stmt, error) { return &fullStmt{}, nil }
func (c *fullConn) Close() error                       { return nil }
func (c *fullConn) Begin() (dbdrv.Tx, error)           { return nil, errors.New("no tx") }

func (c *fullConn) ResetSession(context.Context) error { c.resetCalled = true; return nil }
func (c *fullConn) IsValid() bool                      { return false }
func (c *fullConn) Ping(context.Context) error         { return nil }

func (c *fullConn) CheckNamedValue(nv *dbdrv.NamedValue) error {
	c.checkCalled = true
	nv.Value = "checked"
	return nil
}

func (c *fullConn) PrepareContext(_ context.Context, _ string) (dbdrv.Stmt, error) {
	c.prepareCtxHit = true
	return &fullStmt{}, nil
}

type fullStmt struct{}

func (*fullStmt) Close() error                             { return nil }
func (*fullStmt) NumInput() int                            { return 0 }
func (*fullStmt) Exec([]dbdrv.Value) (dbdrv.Result, error) { return nil, errors.New("no exec") }
func (*fullStmt) Query([]dbdrv.Value) (dbdrv.Rows, error)  { return nil, errors.New("no query") }

// --- a wrapped driver implementing none of them ---

type bareDriver struct{}

func (bareDriver) Open(string) (dbdrv.Conn, error) { return &bareConn{}, nil }

type bareConn struct{}

func (*bareConn) Prepare(string) (dbdrv.Stmt, error) { return &fullStmt{}, nil }
func (*bareConn) Close() error                       { return nil }
func (*bareConn) Begin() (dbdrv.Tx, error)           { return nil, errors.New("no tx") }

func init() {
	sql.Register("optional-full", fullDriver{})
	sql.Register("optional-bare", bareDriver{})
}

// openChaosConn registers a chaos driver over wrapped and returns one raw
// driver.Conn from it, bypassing database/sql's pool so the wrapper type itself
// can be inspected.
func openChaosConn(t *testing.T, chaosName, wrapped string) dbdrv.Conn {
	t.Helper()
	chaossql.Register(chaosName, wrapped, engine.New())
	db, err := sql.Open(chaosName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	c, err := db.Driver().Open("")
	if err != nil {
		t.Fatalf("driver.Open: %v", err)
	}
	return c
}

func TestForwardsOptionalInterfacesWhenPresent(t *testing.T) {
	c := openChaosConn(t, "chaos:optional-full-1", "optional-full")

	sr, ok := c.(dbdrv.SessionResetter)
	if !ok {
		t.Fatal("chaosConn does not implement driver.SessionResetter")
	}
	if err := sr.ResetSession(context.Background()); err != nil {
		t.Errorf("ResetSession: %v", err)
	}

	v, ok := c.(dbdrv.Validator)
	if !ok {
		t.Fatal("chaosConn does not implement driver.Validator")
	}
	if v.IsValid() {
		t.Error("IsValid = true; wrapped conn reports false and that must be forwarded")
	}

	nvc, ok := c.(dbdrv.NamedValueChecker)
	if !ok {
		t.Fatal("chaosConn does not implement driver.NamedValueChecker")
	}
	nv := &dbdrv.NamedValue{Value: "raw"}
	if err := nvc.CheckNamedValue(nv); err != nil {
		t.Errorf("CheckNamedValue: %v", err)
	}
	if nv.Value != "checked" {
		t.Errorf("nv.Value = %v; wrapped checker did not run", nv.Value)
	}

	pc, ok := c.(dbdrv.ConnPrepareContext)
	if !ok {
		t.Fatal("chaosConn does not implement driver.ConnPrepareContext")
	}
	if _, err := pc.PrepareContext(context.Background(), "SELECT 1"); err != nil {
		t.Errorf("PrepareContext: %v", err)
	}
}

func TestOptionalInterfaceFallbacksWhenAbsent(t *testing.T) {
	c := openChaosConn(t, "chaos:optional-bare-1", "optional-bare")

	if err := c.(dbdrv.SessionResetter).ResetSession(context.Background()); err != nil {
		t.Errorf("ResetSession fallback = %v, want nil", err)
	}
	if !c.(dbdrv.Validator).IsValid() {
		t.Error("IsValid fallback = false, want true")
	}
	if err := c.(dbdrv.NamedValueChecker).CheckNamedValue(&dbdrv.NamedValue{}); !errors.Is(err, dbdrv.ErrSkip) {
		t.Errorf("CheckNamedValue fallback = %v, want driver.ErrSkip", err)
	}
	if _, err := c.(dbdrv.ConnPrepareContext).PrepareContext(context.Background(), "SELECT 1"); err != nil {
		t.Errorf("PrepareContext fallback = %v, want nil (falls back to Prepare)", err)
	}
}

func TestStmtForwardsNamedValueChecker(t *testing.T) {
	c := openChaosConn(t, "chaos:optional-full-2", "optional-full")
	s, err := c.Prepare("SELECT 1")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	nvc, ok := s.(dbdrv.NamedValueChecker)
	if !ok {
		t.Fatal("chaosStmt does not implement driver.NamedValueChecker")
	}
	if err := nvc.CheckNamedValue(&dbdrv.NamedValue{}); !errors.Is(err, dbdrv.ErrSkip) {
		t.Errorf("CheckNamedValue = %v, want ErrSkip (fullStmt is not a checker)", err)
	}
}

func TestPingerOnlyWhenWrappedCanPing(t *testing.T) {
	full := openChaosConn(t, "chaos:optional-full-3", "optional-full")
	if p, ok := full.(dbdrv.Pinger); !ok {
		t.Error("wrapping a Pinger produced a conn that is not a Pinger")
	} else if err := p.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}

	bare := openChaosConn(t, "chaos:optional-bare-3", "optional-bare")
	if _, ok := bare.(dbdrv.Pinger); ok {
		t.Error("wrapping a non-Pinger produced a conn claiming driver.Pinger; " +
			"db.Ping() would report healthy without ever pinging")
	}
}
