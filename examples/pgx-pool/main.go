// Command pgx-pool demonstrates pool-level chaos on a pgx pool: a transient
// fault injected into the first Exec, recovered by a retry. Requires a running
// Postgres reachable via DATABASE_URL. Run with `DATABASE_URL=... go run .`.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	pgxpool "github.com/jackc/pgx/v5/pgxpool"

	chaospgx "github.com/RomanAgaltsev/chaotic/adapter/pgx"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpPGX),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("transient"))),
	).Named("pg-flap"))
}

// execWithRetry retries the Exec up to attempts times.
func execWithRetry(ctx context.Context, p *chaospgx.Pool, sql string, attempts int) error {
	var err error
	for range attempts {
		if _, err = p.Exec(ctx, sql); err == nil {
			return nil
		}
	}
	return err
}

func run(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	cp := chaospgx.WrapPool(pool, newEngine())
	if err := execWithRetry(ctx, cp, "SELECT 1", 3); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "exec succeeded after retry despite injected fault")
	return nil
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stdout, "set DATABASE_URL to a running Postgres to run this example")
		return
	}
	if err := run(context.Background(), dsn); err != nil {
		fmt.Fprintln(os.Stderr, "FAILED:", err)
	}
}
