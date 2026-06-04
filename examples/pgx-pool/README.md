# pgx-pool

Pool-level chaos on a pgx pool: a transient fault on the first `Exec`, recovered
by a retry. Requires a running Postgres.

`newEngine` installs a `Times(1)` error rule on `OpPGX`; `execWithRetry` retries,
so the second attempt succeeds. The test is gated behind the `pgxintegration`
build tag (matching the adapter's integration tests) and skips without
`DATABASE_URL`.