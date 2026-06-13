//go:build !chaos_off

package sql

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// errStub is returned by the stub conn/stmt so tests can confirm the chaos
// wrapper delegated to the wrapped driver rather than synthesizing a result.
var errStub = errors.New("stub")

// legacyConn is a minimal driver.Conn (Prepare/Close/Begin) with no context
// interfaces, so it drives the legacy code paths database/sql normally skips.
type legacyConn struct{ beginCalls int }

func (*legacyConn) Prepare(string) (driver.Stmt, error) { return nil, errStub }
func (*legacyConn) Close() error                        { return nil }
func (c *legacyConn) Begin() (driver.Tx, error) {
	c.beginCalls++
	return nil, errStub
}

// legacyPinger adds driver.Pinger to legacyConn.
type legacyPinger struct {
	legacyConn
	pinged bool
}

func (p *legacyPinger) Ping(context.Context) error { p.pinged = true; return nil }

func TestChaosConnBeginDelegates(t *testing.T) {
	inner := &legacyConn{}
	c := &chaosConn{wrapped: inner, eng: engine.New()}
	if _, err := c.Begin(); !errors.Is(err, errStub) {
		t.Fatalf("Begin() err = %v, want errStub", err)
	}
	if inner.beginCalls != 1 {
		t.Fatalf("wrapped Begin called %d times, want 1", inner.beginCalls)
	}
}

func TestChaosConnBeginTxFallsBackToBegin(t *testing.T) {
	// wrapped does not implement driver.ConnBeginTx, so BeginTx must fall back
	// to the legacy Begin.
	inner := &legacyConn{}
	c := &chaosConn{wrapped: inner, eng: engine.New()}
	if _, err := c.BeginTx(context.Background(), driver.TxOptions{}); !errors.Is(err, errStub) {
		t.Fatalf("BeginTx() err = %v, want errStub", err)
	}
	if inner.beginCalls != 1 {
		t.Fatalf("BeginTx did not fall back to Begin (calls=%d)", inner.beginCalls)
	}
}

func TestChaosConnPing(t *testing.T) {
	t.Run("non-pinger is a no-op", func(t *testing.T) {
		c := &chaosConn{wrapped: &legacyConn{}, eng: engine.New()}
		if err := c.Ping(context.Background()); err != nil {
			t.Fatalf("Ping() = %v, want nil", err)
		}
	})
	t.Run("pinger is delegated to", func(t *testing.T) {
		inner := &legacyPinger{}
		c := &chaosConn{wrapped: inner, eng: engine.New()}
		if err := c.Ping(context.Background()); err != nil {
			t.Fatalf("Ping() = %v, want nil", err)
		}
		if !inner.pinged {
			t.Fatal("wrapped Pinger.Ping was not called")
		}
	})
}

// legacyStmt is a minimal driver.Stmt exercising chaosStmt's legacy Query path
// (database/sql uses it when the stmt has no StmtQueryContext).
type legacyStmt struct{ queryCalls int }

func (*legacyStmt) Close() error  { return nil }
func (*legacyStmt) NumInput() int { return 0 }
func (*legacyStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errStub
}
func (s *legacyStmt) Query([]driver.Value) (driver.Rows, error) {
	s.queryCalls++
	return nil, errStub
}

func TestChaosStmtQueryDelegatesWhenChaosDisabled(t *testing.T) {
	inner := &legacyStmt{}
	st := &chaosStmt{wrapped: inner, eng: engine.New(), query: "SELECT 1"}
	if _, err := st.Query(nil); !errors.Is(err, errStub) {
		t.Fatalf("Query() err = %v, want errStub (delegated)", err)
	}
	if inner.queryCalls != 1 {
		t.Fatalf("wrapped Query called %d times, want 1", inner.queryCalls)
	}
}
