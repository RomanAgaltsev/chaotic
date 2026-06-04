//go:build pgxintegration

package pgx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

var sharedPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(runIntegration(m))
}

// runIntegration sets up the shared postgres container + pool, runs the
// tests, and tears everything down. Uses a helper rather than calling
// os.Exit directly so the defers fire.
func runIntegration(m *testing.M) int {
	ctx := context.Background()

	pgC, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("chaotic"),
		tcpostgres.WithUsername("chaotic"),
		tcpostgres.WithPassword("chaotic"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to start postgres container:", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(pgC) }()

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to compute DSN:", err)
		return 1
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to open pgxpool:", err)
		return 1
	}
	defer pool.Close()

	sharedPool = pool
	return m.Run()
}

func newTestPool(t *testing.T, eng *engine.Engine) *Pool {
	t.Helper()
	return WrapPool(sharedPool, eng)
}

func TestIntegrationPoolQueryRoundTrip(t *testing.T) {
	p := newTestPool(t, engine.New())
	var n int
	if err := p.QueryRow(context.Background(), "SELECT 42").Scan(&n); err != nil {
		t.Fatalf("Scan err = %v", err)
	}
	if n != 42 {
		t.Fatalf("got %d, want 42", n)
	}
}

func TestIntegrationPoolExecConnDropReturnsNetOpError(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.ConnDrop()),
	))
	p := newTestPool(t, eng)
	_, err := p.Exec(context.Background(), "SELECT 1")
	if _, ok := errors.AsType[*net.OpError](err); !ok {
		t.Fatalf("Exec err = %v, want *net.OpError", err)
	}
}

func TestIntegrationAcquireConnDropPoisonsTheConn(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchPredicate(func(_ context.Context, op engine.Op) bool {
			// Only fire on the first Query after Acquire, not on Acquire itself.
			return op.Kind == engine.OpPGX && op.Method == "query"
		}),
		engine.Times(1),
		engine.WithFault(fault.ConnDrop()),
	))
	p := newTestPool(t, eng)

	conn, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire err = %v", err)
	}
	// Read out the underlying conn's pointer identity for the post-check.
	rawConn, ok := conn.Unwrap().(*pgxpool.Conn)
	if !ok || rawConn == nil {
		t.Fatalf("Unwrap = %T, want *pgxpool.Conn", conn.Unwrap())
	}
	pgxConnPtr := rawConn.Conn() // *pgx.Conn

	// Fire the chaos.
	_, _ = conn.Query(context.Background(), "SELECT 1")
	conn.Release()

	// The pgx.Conn should now be closed.
	if !pgxConnPtr.IsClosed() {
		t.Fatal("expected the underlying *pgx.Conn to be closed by ConnDrop poison")
	}
}

func TestIntegrationTxRoundTrip(t *testing.T) {
	p := newTestPool(t, engine.New())
	ctx := context.Background()
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin err = %v", err)
	}
	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE chaotic_tx_smoke (id int)"); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("CREATE err = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit err = %v", err)
	}
}

func TestIntegrationCommandTagPassThrough(t *testing.T) {
	p := newTestPool(t, engine.New())
	ctx := context.Background()
	if _, err := p.Exec(ctx, "CREATE TEMP TABLE chaotic_ct (id int)"); err != nil {
		t.Fatalf("CREATE: %v", err)
	}
	tag, err := p.Exec(ctx, "INSERT INTO chaotic_ct VALUES (1), (2), (3)")
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	if tag.RowsAffected() != 3 {
		t.Errorf("RowsAffected = %d, want 3", tag.RowsAffected())
	}
}

// Silence "imported and not used" if pgconn is referenced only via tag.
var (
	_ = pgconn.CommandTag{}
	_ = pgxv5.RepeatableRead
)
