package pgx_test

import (
	"context"
	"errors"

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
