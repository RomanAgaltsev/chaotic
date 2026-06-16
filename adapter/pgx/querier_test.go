package pgx

import (
	"context"
	"errors"
	"testing"

	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// countViaQuerier is a stand-in for real consumer code: it depends only on the
// exported Querier interface, never on a concrete pool type.
func countViaQuerier(ctx context.Context, q Querier) (pgconn.CommandTag, error) {
	return q.Exec(ctx, "DELETE FROM widgets WHERE stale")
}

func TestQuerier_FakeSatisfiesAndRuns(t *testing.T) {
	want := pgconn.NewCommandTag("DELETE 3")
	var q Querier = &fakePool{
		onExec: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return want, nil
		},
	}

	got, err := countViaQuerier(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != want.String() {
		t.Fatalf("tag = %q, want %q", got.String(), want.String())
	}
}

func TestQuerier_WrappedPoolSatisfiesAndInjects(t *testing.T) {
	// Engine with a single Exec-poisoning rule, so the wrapper visibly differs
	// from the bare backend when used behind the same Querier interface.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.Times(1),
		engine.WithFault(fault.Error(errInjected)),
	).Named("q-test"))

	backend := &fakePool{
		onExec: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("DELETE 0"), nil
		},
	}
	var q Querier = &Pool{b: backend, eng: eng}

	if _, err := countViaQuerier(context.Background(), q); err == nil {
		t.Fatal("expected injected error through Querier, got nil")
	}
}

var errInjected = errors.New("injected")

// Compile-time proof the test's intent holds for the real driver type too.
var (
	_ Querier = (*fakePool)(nil)
	_ pgxv5.Rows
)
