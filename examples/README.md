# chaotic examples

Runnable, tested scenarios. Each directory has a `main.go` (`go run .`), a
`main_test.go` (`go test ./...`), and a `README.md`. Per-symbol godoc examples
live next to each package (see the `Example*` functions on
[pkg.go.dev](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic)).

| Scenario | Demonstrates | Adapter |
|----------|--------------|---------|
| [retry-http](retry-http/) | a retry loop recovers from a transient injected failure | adapter/http |
| [circuit-breaker](circuit-breaker/) | a breaker opens after repeated injected failures | adapter/http |
| [db-conn-pool](db-conn-pool/) | the pool evicts a poisoned conn (`ConnDrop` → `ErrBadConn`) | adapter/sql |
| [grpc-stream-reconnect](grpc-stream-reconnect/) | a stream client reconnects after an injected `Unavailable` | adapter/grpc |
| [pgx-pool](pgx-pool/) | pool-level chaos on a pgx pool (integration-gated) | adapter/pgx |
| [redis-cache-fallback](redis-cache-fallback/) | a read-through cache falls back to the DB when Redis fails | adapter/redis |
| [kafka-write-retry](kafka-write-retry/) | a producer retries through a transient Kafka write outage (needs Docker) | adapter/kafka |
| [nats-request-retry](nats-request-retry/) | a request/reply caller retries through a transient NATS outage | adapter/nats |
| [mongo-read-fallback](mongo-read-fallback/) | a read retries through a transient MongoDB step-down (needs Docker) | adapter/mongo |
| [rabbitmq-publish-retry](rabbitmq-publish-retry/) | a publisher retries through a transient RabbitMQ outage (needs Docker) | adapter/rabbitmq |
| [aws-dynamodb-retry](aws-dynamodb-retry/) | the AWS SDK's own retryer recovers from an injected outage | adapter/aws |
| [net-conn-drop](net-conn-drop/) | a read loop retries through a transient connection drop | adapter/net |
| [chaos-point](chaos-point/) | an explicit `chaos.Point` guards a post-commit hook | chaos |
| [clock-skew](clock-skew/) | a token expires once `fault.Clock` skews `engine.Now` past its TTL | fault.Clock |
| [terms-dsl](terms-dsl/) | a one-line terms string activates chaos with no rule-building code | source/terms |
| [prod-safety-rails](prod-safety-rails/) | failure budget + caps + guard + kill switch bound the blast radius | engine |

Examples that wrap a third-party adapter (`adapter/grpc`, `pgx`, `redis`,
`kafka`, `nats`, `mongo`, `rabbitmq`, `aws`) are each their own Go module, so
running them pulls in only that integration's dependencies.

Most examples run with no external services — in-process fakes (`miniredis`, an
embedded `nats-server`, `net.Pipe`) or an `httptest` server stand in. Four need a
live backend and **skip** when it is unavailable:

- `kafka-write-retry`, `mongo-read-fallback`, `rabbitmq-publish-retry` start a
  real broker via [testcontainers-go](https://golang.testcontainers.org/), so
  **Docker must be running**.
- `pgx-pool` needs a reachable Postgres.
