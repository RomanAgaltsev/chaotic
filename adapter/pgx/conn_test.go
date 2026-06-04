package pgx

import (
	"context"
	"errors"
	"testing"

	pgxv5 "github.com/jackc/pgx/v5"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newConnWithFake(eng *engine.Engine, f *fakeConn) *Conn {
	return &Conn{
		b:   f,
		eng: eng,
	}
}

func TestConnQueryPassThroughWhenEngineEmpty(t *testing.T) {
	called := false
	f := &fakeConn{
		onQuery: func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
			called = true
			return nil, nil
		},
	}
	c := newConnWithFake(engine.New(), f)
	if _, err := c.Query(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("Query err = %v", err)
	}
	if !called {
		t.Fatal("expected underlying Query")
	}
}

func TestConnQueryReturnsChaosError(t *testing.T) {
	want := errors.New("chaos")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	c := newConnWithFake(eng, &fakeConn{})
	_, err := c.Query(context.Background(), "SELECT 1")
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestConnQueryAttrTxFalse(t *testing.T) {
	got := ""
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchPredicate(func(_ context.Context, op engine.Op) bool {
			got = op.Attrs["tx"]
			return op.Kind == engine.OpPGX
		}),
		engine.WithFault(fault.Error(errors.New("e"))),
	))
	c := newConnWithFake(eng, &fakeConn{})
	_, _ = c.Query(context.Background(), "SELECT 1")
	if got != "false" {
		t.Fatalf("Attrs[tx] = %q, want false", got)
	}
}

func TestConnQueryRowReturnsChaosRow(t *testing.T) {
	want := errors.New("chaos")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	c := newConnWithFake(eng, &fakeConn{})
	row := c.QueryRow(context.Background(), "SELECT 1")
	if got := row.Scan(); !errors.Is(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestConnSendBatchReturnsChaosBatch(t *testing.T) {
	want := errors.New("chaos")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	c := newConnWithFake(eng, &fakeConn{})
	br := c.SendBatch(context.Background(), &pgxv5.Batch{})
	if got := br.Close(); !errors.Is(got, want) {
		t.Fatalf("Close = %v, want %v", got, want)
	}
}

func TestConnBeginReturnsWrappedTx(t *testing.T) {
	innerTx := &fakeTx{}
	f := &fakeConn{
		onBegin: func(context.Context) (pgxv5.Tx, error) { return innerTx, nil },
	}
	c := newConnWithFake(engine.New(), f)
	tx, err := c.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin err = %v", err)
	}
	if tx == nil {
		t.Fatal("Begin returned nil Tx")
	}
	// Operations on the returned *Tx route through chaos when configured.
	if tx.b != pgxv5.Tx(innerTx) {
		t.Errorf("returned *Tx does not wrap the underlying tx")
	}
}

func TestConnBeginChaosShortCircuits(t *testing.T) {
	want := errors.New("chaos")
	called := false
	f := &fakeConn{
		onBegin: func(context.Context) (pgxv5.Tx, error) { called = true; return nil, nil },
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	c := newConnWithFake(eng, f)
	tx, err := c.Begin(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if tx != nil {
		t.Fatal("expected nil Tx on chaos")
	}
	if called {
		t.Fatal("underlying Begin should not be called")
	}
}

func TestConnBeginTxPassesOptionsAndCarriesIso(t *testing.T) {
	gotIso := ""
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchPredicate(func(_ context.Context, op engine.Op) bool {
			gotIso = op.Attrs["iso_level"]
			return op.Kind == engine.OpPGX
		}),
		engine.WithFault(fault.Error(errors.New("e"))),
	))
	c := newConnWithFake(eng, &fakeConn{})
	_, _ = c.BeginTx(context.Background(), pgxv5.TxOptions{IsoLevel: pgxv5.RepeatableRead})
	if gotIso != "repeatable read" {
		t.Fatalf("Attrs[iso_level] = %q, want %q", gotIso, "repeatable read")
	}
}

func TestConnPingIsPassThrough(t *testing.T) {
	called := false
	f := &fakeConn{
		onPing: func(context.Context) error {
			called = true
			return nil
		},
	}
	// Even with a chaos rule, Ping must NOT be intercepted.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(errors.New("should not fire"))),
	))
	c := newConnWithFake(eng, f)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping err = %v", err)
	}
	if !called {
		t.Fatal("expected underlying Ping")
	}
}

func TestConnReleaseIsNoOpForStandaloneVariant(t *testing.T) {
	// Build a *Conn whose raw is nil (no pgxpool.Conn underneath).
	c := &Conn{
		b:   &fakeConn{},
		eng: engine.New(),
		raw: nil,
	}
	// Should not panic. Returns nothing.
	c.Release()
}

func TestConnUnwrapReturnsRaw(t *testing.T) {
	c := &Conn{
		b:   &fakeConn{},
		eng: engine.New(),
		raw: "sentinel",
	}
	if got := c.Unwrap(); got != "sentinel" {
		t.Fatalf("Unwrap = %v, want %q", got, "sentinel")
	}
}

func TestConnConnDropClosesUnderlyingConn(t *testing.T) {
	closed := false
	f := &fakeConn{
		onClose: func(context.Context) error { closed = true; return nil },
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.ConnDrop()),
	))
	c := newConnWithFake(eng, f)
	_, _ = c.Query(context.Background(), "SELECT 1")
	if !closed {
		t.Fatal("expected ConnDrop to close the underlying conn")
	}
}

func TestConnNonConnDropDoesNotClose(t *testing.T) {
	closed := false
	f := &fakeConn{
		onClose: func(context.Context) error { closed = true; return nil },
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(errors.New("not a conn drop"))),
	))
	c := newConnWithFake(eng, f)
	_, _ = c.Query(context.Background(), "SELECT 1")
	if closed {
		t.Fatal("non-ConnDrop chaos must not close the conn")
	}
}
