package pgx

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newPoolWithFake(eng *engine.Engine, f *fakePool) *Pool {
	return &Pool{
		b:   f,
		eng: eng,
	}
}

func TestPoolQueryPassThroughWhenEngineEmpty(t *testing.T) {
	called := false
	f := &fakePool{
		onQuery: func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
			called = true
			return nil, nil
		},
	}
	p := newPoolWithFake(engine.New(), f)
	if _, err := p.Query(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("Query err = %v", err)
	}
	if !called {
		t.Fatal("expected underlying Query to be called")
	}
}

func TestPoolQueryReturnsChaosError(t *testing.T) {
	want := errors.New("chaos")
	called := false
	f := &fakePool{
		onQuery: func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
			called = true
			return nil, nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	p := newPoolWithFake(eng, f)
	_, err := p.Query(context.Background(), "SELECT * FROM users")
	if !errors.Is(err, want) {
		t.Fatalf("Query err = %v, want %v", err, want)
	}
	if called {
		t.Fatal("underlying Query should not be called when chaos fires")
	}
}

func TestPoolExecPassThroughWhenEngineEmpty(t *testing.T) {
	called := false
	f := &fakePool{
		onExec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			called = true
			return pgconn.CommandTag{}, nil
		},
	}
	p := newPoolWithFake(engine.New(), f)
	if _, err := p.Exec(context.Background(), "INSERT INTO t VALUES(1)"); err != nil {
		t.Fatalf("Exec err = %v", err)
	}
	if !called {
		t.Fatal("expected underlying Exec to be called")
	}
}

func TestPoolExecConnDropTranslatesToNetOpError(t *testing.T) {
	f := &fakePool{
		onExec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.ConnDrop()),
	))
	p := newPoolWithFake(eng, f)
	_, err := p.Exec(context.Background(), "DELETE FROM t WHERE id = $1", 1)
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("Exec err = %v, want *net.OpError", err)
	}
	if !errors.Is(opErr.Err, io.ErrUnexpectedEOF) {
		t.Errorf("OpError.Err = %v, want io.ErrUnexpectedEOF", opErr.Err)
	}
}

func TestWrapPoolNilEnginePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil engine")
		}
	}()
	// We don't have a real *pgxpool.Pool here, but WrapPool checks the engine
	// before the pool, so nil/zero pool is irrelevant for this assertion.
	_ = WrapPool(nil, nil)
}

func TestPoolQueryRowReturnsChaosRowOnFault(t *testing.T) {
	want := errors.New("chaos")
	f := &fakePool{
		onQueryRow: func(ctx context.Context, sql string, args ...any) pgxv5.Row {
			t.Fatal("underlying QueryRow should not be called when chaos fires")
			return nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	p := newPoolWithFake(eng, f)
	row := p.QueryRow(context.Background(), "SELECT 1 FROM dual")
	if got := row.Scan(); !errors.Is(got, want) {
		t.Fatalf("Scan err = %v, want %v", got, want)
	}
}

func TestPoolQueryRowPassThroughWhenEngineEmpty(t *testing.T) {
	called := false
	f := &fakePool{
		onQueryRow: func(ctx context.Context, sql string, args ...any) pgxv5.Row {
			called = true
			return chaosRow{} // any non-nil pgx.Row
		},
	}
	p := newPoolWithFake(engine.New(), f)
	_ = p.QueryRow(context.Background(), "SELECT 1")
	if !called {
		t.Fatal("expected underlying QueryRow to be called")
	}
}

func TestPoolSendBatchReturnsChaosBatchOnFault(t *testing.T) {
	want := errors.New("chaos")
	f := &fakePool{
		onSendBatch: func(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
			t.Fatal("underlying SendBatch should not be called when chaos fires")
			return nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	p := newPoolWithFake(eng, f)
	br := p.SendBatch(context.Background(), &pgxv5.Batch{})
	if got := br.Close(); !errors.Is(got, want) {
		t.Fatalf("Close err = %v, want %v", got, want)
	}
}

func TestPoolSendBatchPassThroughWhenEngineEmpty(t *testing.T) {
	called := false
	f := &fakePool{
		onSendBatch: func(ctx context.Context, b *pgxv5.Batch) pgxv5.BatchResults {
			called = true
			return chaosBatch{}
		},
	}
	p := newPoolWithFake(engine.New(), f)
	_ = p.SendBatch(context.Background(), &pgxv5.Batch{})
	if !called {
		t.Fatal("expected underlying SendBatch to be called")
	}
}

func TestPoolPingIsAlwaysPassThrough(t *testing.T) {
	called := false
	f := &fakePool{
		onPing: func(ctx context.Context) error { called = true; return nil },
	}
	// Even with a rule that matches OpPGX, Ping should NOT fire chaos.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(errors.New("should not fire"))),
	))
	p := newPoolWithFake(eng, f)
	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("Ping err = %v", err)
	}
	if !called {
		t.Fatal("expected underlying Ping to be called")
	}
}

func TestPoolBeginReturnsWrappedTx(t *testing.T) {
	innerTx := &fakeTx{}
	f := &fakePool{
		onBegin: func(context.Context) (pgxv5.Tx, error) { return innerTx, nil },
	}
	p := newPoolWithFake(engine.New(), f)
	tx, err := p.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin err = %v", err)
	}
	if tx == nil {
		t.Fatal("Begin returned nil")
	}
	if tx.b != pgxv5.Tx(innerTx) {
		t.Error("returned *Tx does not wrap underlying tx")
	}
}

func TestPoolBeginChaosShortCircuits(t *testing.T) {
	want := errors.New("chaos")
	f := &fakePool{
		onBegin: func(context.Context) (pgxv5.Tx, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	p := newPoolWithFake(eng, f)
	tx, err := p.Begin(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if tx != nil {
		t.Fatal("expected nil Tx on chaos")
	}
}

func TestPoolAcquireChaosShortCircuits(t *testing.T) {
	want := errors.New("chaos")
	f := &fakePool{
		onAcquire: func(context.Context) (*pgxpool.Conn, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(want)),
	))
	p := newPoolWithFake(eng, f)
	c, err := p.Acquire(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if c != nil {
		t.Fatal("expected nil Conn on chaos")
	}
}
