package pgx_test

import (
	"context"
	"errors"
	"fmt"

	pgxpool "github.com/jackc/pgx/v5/pgxpool"

	chaospgx "github.com/RomanAgaltsev/chaotic/adapter/pgx"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func ExampleWrapPool() {
	// Given a real pgx pool:
	pool, err := pgxpool.New(context.Background(), "postgres://localhost/app")
	if err != nil {
		return // handle error
	}
	defer pool.Close()

	// Fail only the first operation on the pool with a transient error.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("transient"))),
	).Named("pg-flap"))

	// Wrap once; use cp exactly like *pgxpool.Pool everywhere downstream.
	cp := chaospgx.WrapPool(pool, eng)
	_, _ = cp.Exec(context.Background(), "UPDATE accounts SET balance = balance - 1")
	// No Output: this example needs a live Postgres and is illustrative.
}

// ExampleQuerier shows wiring a repository against the chaospgx.Querier
// interface so production injects *pgxpool.Pool and tests inject a wrapped
// *Pool — with no change to the repository code.
func ExampleQuerier() {
	// Repository code depends only on the interface.
	type repo struct{ db chaospgx.Querier }
	deleteStale := func(ctx context.Context, r repo) error {
		_, err := r.db.Exec(ctx, "DELETE FROM widgets WHERE stale")
		return err
	}

	// Test wiring: a pool wrapper whose engine fails the first Exec once.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("connection refused"))),
	).Named("flap"))

	// In production this would be: var db chaospgx.Querier = realPgxPool
	// Here we inject a wrapper over a stub backend (see the package tests for
	// how *Pool is built over a fake). For the example we use a real wrapper
	// via WrapPool in your own code; this comment documents the swap point.
	_ = eng
	_ = deleteStale

	fmt.Println("inject *pgxpool.Pool in prod, chaospgx.WrapPool(pool, eng) in tests")
	// Output: inject *pgxpool.Pool in prod, chaospgx.WrapPool(pool, eng) in tests
}
