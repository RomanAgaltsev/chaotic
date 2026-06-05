package pgx

import (
	"context"
	"errors"
	"testing"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each call, or chaos silently
// stops once the cap is reached. The fault here returns nil from Before
// (latency), so the slot is freed only by the adapter running After.
func TestPoolReleasesMaxConcurrentSlot(t *testing.T) {
	f := &fakePool{
		onQuery: func(ctx context.Context, sql string, args ...any) (pgxv5.Rows, error) {
			return nil, nil
		},
	}
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))
	p := newPoolWithFake(eng, f)
	for range 3 {
		_, _ = p.Query(context.Background(), "SELECT 1")
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential calls; the max-concurrent slot is leaking", got)
	}
}

// The same guarantee for *Conn (it routes through runChaos).
func TestConnReleasesMaxConcurrentSlot(t *testing.T) {
	f := &fakeConn{
		onExec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, nil
		},
	}
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))
	c := &Conn{b: f, eng: eng}
	for range 3 {
		_, _ = c.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential calls; the max-concurrent slot is leaking", got)
	}
}

// Injected pgx outcomes must reach the failure budget. With a 50%% budget over a
// 2-call window and an always-firing error rule, the budget fills after two
// injected errors and suppresses the third call. If outcomes were never
// reported the window would never fill and all three would fire.
func TestPoolReportsOutcomesToFailureBudget(t *testing.T) {
	backendCalls := 0
	f := &fakePool{
		onExec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			backendCalls++
			return pgconn.CommandTag{}, nil
		},
	}
	eng := engine.New(engine.WithFailureBudget(0.5, 2)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.WithFault(fault.Error(errors.New("boom"))),
	).Named("budget"))
	p := newPoolWithFake(eng, f)
	for range 3 {
		_, _ = p.Exec(context.Background(), "DELETE FROM t")
	}
	if got := eng.Hits("budget"); got != 2 {
		t.Fatalf("rule fired %d times, want 2 (3rd call suppressed by budget); outcomes not reported?", got)
	}
	if backendCalls != 1 {
		t.Fatalf("backend called %d times, want 1 (only the budget-suppressed call reaches it)", backendCalls)
	}
}
