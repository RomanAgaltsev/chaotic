# chaotic cookbook

"How do I test that…?" — each recipe maps a resilience question to the rule shape
that answers it and the runnable [example](../examples/) that proves it. Recipes
without a dedicated example are answered inline below.

New to chaotic? Start with the main [README](../README.md) "Core concepts" and the
`chaostest` / `chaostest/quick` / `chaostest/scenarios` helpers.

## Recipes

| # | How do I test that… | Where |
|---|---------------------|-------|
| 1 | my retry loop fires on transient HTTP failures? | [retry-http](../examples/retry-http/) |
| 2 | my circuit breaker opens after N consecutive failures? | [circuit-breaker](../examples/circuit-breaker/) |
| 3 | my timeout fires when an upstream is slow? | [#timeout-on-slow-upstream](#timeout-on-slow-upstream) |
| 4 | my retry budget is bounded? | [#bounded-retry-budget](#bounded-retry-budget) |
| 5 | a fan-out fails partially and degrades gracefully? | [fanout-partial-failure](../examples/fanout-partial-failure/) |
| 6 | my rate limiter sheds load under chaos? | [#rate-limiter-sheds-load](#rate-limiter-sheds-load) |
| 7 | my pg connection pool evicts a poisoned conn? | [db-conn-pool](../examples/db-conn-pool/), [pgx-pool](../examples/pgx-pool/) |
| 8 | gRPC stream reconnection works? | [grpc-stream-reconnect](../examples/grpc-stream-reconnect/) |
| 9 | S3 backoff respects jitter? | [aws-dynamodb-retry](../examples/aws-dynamodb-retry/) |
| 10 | Redis cache fallback to DB works? | [redis-cache-fallback](../examples/redis-cache-fallback/) |
| 11 | my Kafka consumer recovers after a broker outage? | [kafka-write-retry](../examples/kafka-write-retry/) |
| 12 | NATS reconnection delivers buffered messages? | [nats-request-retry](../examples/nats-request-retry/) |
| 13 | my distributed lock fails closed? | [#distributed-lock-fails-closed](#distributed-lock-fails-closed) |
| 14 | webhook delivery retries respect upstream `Retry-After`? | [#webhook-respects-retry-after](#webhook-respects-retry-after) |
| 15 | pagination handles a mid-stream truncation? | [slow-body-read](../examples/slow-body-read/) |
| 16 | timezone arithmetic survives clock skew? | [clock-skew](../examples/clock-skew/) |
| 17 | JSON parsing handles a body truncated mid-object? | [slow-body-read](../examples/slow-body-read/), [response-mutate](../examples/response-mutate/) |
| 18 | protocol buffers handle a body truncated mid-message? | [#protobuf-truncated-mid-message](#protobuf-truncated-mid-message) |
| 19 | retry storms are bounded? | [#bounded-retry-storms](#bounded-retry-storms) |
| 20 | observability emits proper logs/traces during a chaos fire? | [observability-during-chaos](../examples/observability-during-chaos/) |

## Inline recipes

### timeout-on-slow-upstream

Inject latency longer than the caller's deadline and assert the call times out.

```go
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpHTTPClient),
    engine.WithFault(fault.Latency(2*time.Second)),
).Named("slow"))
// Caller uses a 200ms context deadline; assert the GET returns context.DeadlineExceeded.
```

Nearest runnable example: retry-http.

### bounded-retry-budget

Cap how many faults fire across a sliding window with engine.WithFailureBudget, so a retrying client cannot be starved indefinitely.

```go
eng := engine.New(engine.WithFailureBudget(0.5, 10)) // stop injecting once >50% of last 10 calls errored
```

Nearest runnable example: prod-safety-rails.

### rate-limiter-sheds-load

Fault a fraction of calls and assert your limiter sheds (rejects/queues) rather than melting down.

```go
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpHTTPClient),
    engine.Probability(0.2, 42),
    engine.WithFault(fault.Error(errors.New("overloaded"))),
).Named("shed"))
```

Nearest runnable example: prod-safety-rails.

### distributed-lock-fails-closed

Fault the Redis call backing your lock and assert the lock acquisition fails closed (denies) rather than open (grants on error).

```go
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpRedis),
    engine.MatchName("SET"),
    engine.WithFault(fault.ConnDrop()),
).Named("lock-redis-down"))
```

Nearest runnable example: redis-cache-fallback.

### webhook-respects-retry-after

Return a 429 with a Retry-After header and assert your delivery loop honors it.

```go
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpHTTPClient),
    engine.WithFault(fault.HTTPStatus(429)),
    engine.WithFault(fault.HeaderInject("Retry-After", "2")),
).Named("throttled"))
```

Nearest runnable example: retry-http.

### protobuf-truncated-mid-message

Truncate a streamed/proto response body and assert your decoder surfaces a clean error instead of panicking.

```go
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpIO),
    engine.WithFault(fault.Truncate(8)),
).Named("trunc"))
```

Nearest runnable example: slow-body-read.

### bounded-retry-storms

Combine WithRateLimit and WithMaxConcurrent so injected failures cannot amplify into an unbounded retry storm.

```go
eng := engine.New(
    engine.WithRateLimit(50),     // at most 50 fault fires/sec
    engine.WithMaxConcurrent(10), // at most 10 in-flight faulted calls
)
```

Nearest runnable example: prod-safety-rails.
