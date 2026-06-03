# chaotic examples

Runnable, tested scenarios. Each directory has a `main.go` (`go run .`), a
`main_test.go` (`go test ./...`), and a `README.md`. Per-symbol godoc examples
live next to each package (see the `Example*` functions on
[pkg.go.dev](https://pkg.go.dev/github.com/ag4r/chaotic)).

| Scenario | Demonstrates | Adapter |
|----------|--------------|---------|
| [retry-http](retry-http/) | a retry loop recovers from a transient injected failure | adapter/http |
| [circuit-breaker](circuit-breaker/) | a breaker opens after repeated injected failures | adapter/http |
| [db-conn-pool](db-conn-pool/) | the pool evicts a poisoned conn (`ConnDrop` → `ErrBadConn`) | adapter/sql |
| [grpc-stream-reconnect](grpc-stream-reconnect/) | a stream client reconnects after an injected `Unavailable` | adapter/grpc |
| [chaos-point](chaos-point/) | an explicit `chaos.Point` guards a post-commit hook | chaos (v3) |
| [prod-safety-rails](prod-safety-rails/) | failure budget + caps + guard + kill switch bound the blast radius | engine |
| [pgx-pool](pgx-pool/) | pool-level chaos on a pgx pool (integration-gated) | adapter/pgx |

`grpc-stream-reconnect`, `chaos-point`, and `pgx-pool` are added by their
respective plans; the rest ship with the v3 plan.