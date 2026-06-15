# chaotic

[![test](https://github.com/RomanAgaltsev/chaotic/actions/workflows/test.yml/badge.svg)](https://github.com/RomanAgaltsev/chaotic/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/RomanAgaltsev/chaotic/branch/main/graph/badge.svg)](https://codecov.io/gh/RomanAgaltsev/chaotic)
[![Go Reference](https://pkg.go.dev/badge/github.com/RomanAgaltsev/chaotic.svg)](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic)
[![Go Report Card](https://goreportcard.com/badge/github.com/RomanAgaltsev/chaotic)](https://goreportcard.com/report/github.com/RomanAgaltsev/chaotic)
[![Release](https://img.shields.io/github/v/release/RomanAgaltsev/chaotic)](https://github.com/RomanAgaltsev/chaotic/releases)
[![License: MIT](https://img.shields.io/github/license/RomanAgaltsev/chaotic)](./LICENSE)

**In-process chaos engineering for Go.** Wrap your integration boundaries — `http.RoundTripper`, `net/http` middleware, `database/sql` driver, gRPC interceptors, `pgx` pool, Redis, Kafka, NATS, MongoDB, RabbitMQ, the AWS SDK, raw `net.Conn`, and any `io.Reader`/`io.Writer` — and inject latency, errors, panics, connection drops, clock skew, slow or truncated streams, and HTTP faults on demand. When no rules are configured the wrappers are near-zero-cost passthroughs, so chaotic is safe to leave linked everywhere. For a hard guarantee, the `chaos_off` build tag compiles every wrapper down to a bare passthrough.

```go
eng := chaostest.New(t)
eng.AddRule(engine.NewRule(
    engine.MatchKind(engine.OpHTTPClient),
    engine.MatchName("/users/*"),
    engine.Times(2),
    engine.WithFaults(fault.Latency(200*time.Millisecond), fault.Error(io.ErrUnexpectedEOF)),
).Named("transient-failure"))

client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
// ... exercise code under test; it should retry and recover ...

chaostest.AssertHits(t, eng, "transient-failure", 2)
```

## Why chaotic?

- **In-process, not a sidecar.** No proxy, no toxiproxy container, no network rewiring. Chaos is injected exactly where your code touches a dependency.
- **Test-first, production-ready.** Drive deterministic failures from a unit test, or hot-load rules into a running service from a YAML file or an admin HTTP endpoint.
- **Near-zero overhead when idle.** Adapters check `Engine.Enabled()` before allocating anything; with no rules the path is a passthrough. The `chaos_off` build tag removes the machinery entirely.
- **Safe by construction.** Failure budgets, rate limits, concurrency caps, a production guard, a kill switch, and a rule linter bound the blast radius.
- **Dependency-light core.** The engine, faults, the terms DSL, and the stdlib adapters use only the standard library. Heavier integrations (gRPC, pgx, Redis, Kafka, NATS, MongoDB, RabbitMQ, AWS, Prometheus, OTel) live in separate modules so you pull in only what you use.

## Install

```bash
go get github.com/RomanAgaltsev/chaotic
```

The core module ships the engine, faults, the terms DSL, and the stdlib adapters (HTTP client, HTTP server, `database/sql`, raw `net.Conn`). Integrations that pull in third-party dependencies are separate modules — install the ones you need:

```bash
go get github.com/RomanAgaltsev/chaotic/adapter/grpc         # gRPC interceptors
go get github.com/RomanAgaltsev/chaotic/adapter/pgx          # jackc/pgx v5 pool & conn
go get github.com/RomanAgaltsev/chaotic/adapter/redis        # redis/go-redis v9 hook
go get github.com/RomanAgaltsev/chaotic/adapter/kafka        # segmentio/kafka-go writer & reader
go get github.com/RomanAgaltsev/chaotic/adapter/nats         # nats.go connection
go get github.com/RomanAgaltsev/chaotic/adapter/mongo        # mongo-driver v2 collection/db/client
go get github.com/RomanAgaltsev/chaotic/adapter/rabbitmq     # amqp091-go channel & connection
go get github.com/RomanAgaltsev/chaotic/adapter/aws          # aws-sdk-go-v2 middleware
go get github.com/RomanAgaltsev/chaotic/adapter/io           # io.Reader / io.Writer stream faults
go get github.com/RomanAgaltsev/chaotic/observer/slog        # slog observer
go get github.com/RomanAgaltsev/chaotic/observer/prometheus  # Prometheus metrics
go get github.com/RomanAgaltsev/chaotic/observer/otel        # OpenTelemetry
go get github.com/RomanAgaltsev/chaotic/source/file          # YAML rule files (live reload)
go get github.com/RomanAgaltsev/chaotic/source/http          # admin HTTP endpoint
```

The raw-`net.Conn` (`adapter/net`) and `io.Reader`/`io.Writer` (`adapter/io`) adapters are their own modules too, but they depend only on the standard library.

Requires Go 1.26+.

## Core concepts

chaotic has three moving parts:

1. **Engine** (`engine.Engine`) — holds the rules and decides, on every intercepted call, whether to inject a fault. Safe for concurrent use; rule swaps are copy-on-write.
2. **Rules** (`engine.Rule`) — a *match → counter → faults* triple. A rule matches certain operations, a counter decides which matches actually fire, and the faults are what gets injected.
3. **Adapters** — thin wrappers around an integration boundary that build an `Op`, ask the engine to `Eval` it, and apply whatever `Action` comes back.

```go
eng := engine.New().AddRule(engine.NewRule(
    engine.MatchKind(engine.OpHTTPClient), // match: HTTP client calls
    engine.MatchName("/checkout"),         // match: path glob
    engine.Probability(0.1, 42),           // counter: ~10% of matches, seeded
    engine.WithFault(fault.Latency(2*time.Second)), // fault: 2s of added latency
).Named("slow-checkout"))
```

## Adapters

Each adapter wraps one integration boundary and populates a stable, low-cardinality `Op` so observer labels stay consistent.

| Adapter | Wrap with | `Op.Kind` | `Op.Name` | `Op.Method` | `Op.Attrs` |
|---------|-----------|-----------|-----------|-------------|------------|
| HTTP client | `adapter/http`: `WrapTransport(rt, eng)` | `OpHTTPClient` | request path | HTTP method | `host` |
| HTTP server | `adapter/httpsrv`: `httpsrv.Middleware(eng)` | `OpHTTPServer` | request path | HTTP method | `remote` |
| `database/sql` | `adapter/sql`: `Register(name, wrapped, eng)` | `OpSQL` | statement class | statement class | `query` |
| gRPC | `adapter/grpc`: `Unary/StreamClientInterceptor`, `Unary/StreamServerInterceptor` | `OpGRPCClient` / `OpGRPCServer` | full method | `unary` / `stream` | — |
| pgx (v5, native) | `adapter/pgx`: `WrapPool(p, eng)`, `WrapConn(c, eng)` | `OpPGX` | statement class | statement class | `query` |
| Redis (go-redis v9) | `adapter/redis`: `NewHook(eng)` | `OpRedis` | command name | `single` / `pipeline` / `dial` | `key` |
| Kafka (segmentio) | `adapter/kafka`: `WrapWriter(w, eng)`, `WrapReader(r, eng)` | `OpKafka` | topic | `write` / `read` | — |
| NATS (nats.go) | `adapter/nats`: `WrapConn(nc, eng)`, `Option(eng)` | `OpNATS` | subject | `publish` / `request` / … | `queue` |
| MongoDB (driver v2) | `adapter/mongo`: `WrapCollection/Database/Client(x, eng)` | `OpMongo` | command name | `command` | `db`, `coll` |
| RabbitMQ (amqp091) | `adapter/rabbitmq`: `WrapChannel(ch, eng)`, `WrapConnection(c, eng)` | `OpRabbitMQ` | routing key / queue | `publish` / `consume` | `exchange` / `queue` |
| AWS SDK (v2) | `adapter/aws`: `AppendChaosMiddleware(&cfg, eng)` | `OpAWS` | `service.operation` | `request` | `service`, `operation`, `region` |
| raw `net.Conn` | `adapter/net`: `WrapConn/WrapListener/WrapDialer` | `OpNet` | conn name / address | `read` / `write` / `dial` | `network` |
| `io.Reader` / `io.Writer` | `adapter/io`: `WrapReader(r, eng)`, `WrapWriter(w, eng)` | `OpIO` | — | `read` / `write` | — |
| explicit point | `chaos`: `Point(ctx, name)` | `OpExplicit` | the point name | — | caller-supplied |

Keep custom attribute values low-cardinality: they can become metric labels, and unbounded values (raw SQL text, user IDs) blow up an observer's memory.

### HTTP client

```go
client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
```

### HTTP server

```go
mux := http.NewServeMux()
// ... register handlers ...
srv := httpsrv.Middleware(eng)(mux)
```

### database/sql

Register a chaos driver that delegates to an already-registered driver, then open the chaos driver:

```go
chaossql.Register("chaos:postgres", "pgx", eng)
db, err := sql.Open("chaos:postgres", dsn)
```

### gRPC

```go
conn, err := grpc.NewClient(target,
    grpc.WithUnaryInterceptor(chaosgrpc.UnaryClientInterceptor(eng)),
    grpc.WithStreamInterceptor(chaosgrpc.StreamClientInterceptor(eng)),
)
```

### pgx (native, v5)

```go
chaosPool := chaospgx.WrapPool(pool, eng) // intercepts Query, Exec, SendBatch, Begin, Acquire, ...
```

### Redis (go-redis v9)

```go
client := goredis.NewClient(opts)
client.AddHook(chaosredis.NewHook(eng)) // hooks single commands, pipelines, and dials
```

### Kafka (segmentio/kafka-go)

```go
w := chaoskafka.WrapWriter(&kafkago.Writer{ /* ... */ }, eng) // WriteMessages
r := chaoskafka.WrapReader(kafkago.NewReader(cfg), eng)       // ReadMessage / FetchMessage
```

### NATS (nats.go)

```go
nc, err := natsgo.Connect(url, chaosnats.Option(eng)) // or chaosnats.WrapConn(nc, eng)
```

### MongoDB (driver v2)

```go
coll := chaosmongo.WrapCollection(db.Collection("orders"), eng) // also WrapDatabase / WrapClient
```

### RabbitMQ (amqp091-go)

```go
ch := chaosrabbitmq.WrapChannel(amqpChannel, eng) // Publish / Consume; also WrapConnection
```

### AWS SDK for Go v2

```go
cfg, _ := config.LoadDefaultConfig(ctx)
chaosaws.AppendChaosMiddleware(&cfg, eng) // every SDK client built from cfg is now faultable
```

### Raw net.Conn

```go
conn = chaosnet.WrapConn(conn, eng)        // Read / Write
ln = chaosnet.WrapListener(ln, eng)        // faults accepted conns
dialer := chaosnet.WrapDialer(eng, net.Dial) // faults at dial time
```

### Raw io.Reader / io.Writer

Wrap any stream — an HTTP response body, a file, a buffer — to inject the
stream-shaping faults (`SlowReader`/`SlowWriter`/`Truncate`) or any ordinary
fault:

```go
r = chaosio.WrapReader(resp.Body, eng) // throttle or truncate reads
w = chaosio.WrapWriter(dst, eng)       // throttle or truncate writes
```

### Explicit chaos points

For places no adapter reaches — between two pure functions, inside a goroutine, at a state-machine transition — bind an engine to the context and drop a `chaos.Point`:

```go
ctx = chaos.WithEngine(ctx, eng)
// ... downstream ...
if err := chaos.Point(ctx, "checkout.afterCommit"); err != nil {
    return err
}
```

A point on a context with no engine (or a disabled engine) is a silent, allocation-free no-op, so points are safe to leave in production code.

## Faults

`fault.Fault` is one chaos primitive. The built-ins:

| Fault | Effect |
|-------|--------|
| `Latency(d)` | sleep `d` (respects context cancellation) |
| `Jittered(min, max)` | sleep a uniform random duration in `[min, max]` |
| `JitteredSeed(min, max, seed)` | seeded, reproducible jitter |
| `Error(err)` | return `err` in the adapter's native error model |
| `Panic(v)` | `panic(v)` through the wrapped call |
| `ConnDrop()` | substitute the adapter's native connection-drop error (`driver.ErrBadConn`, gRPC `Unavailable`, …); models a hard reset |
| `Disconnect()` | substitute the adapter's native *orderly*-close error (a TCP FIN); distinct from `ConnDrop` because clients often handle a clean close and a hard reset differently |
| `HTTPStatus(code, body…)` | make the HTTP adapters render a specific status |
| `HeaderInject(key, value)` | set a header flowing toward the code under test |
| `HeaderStrip(key)` | delete a header flowing toward the code under test |
| `Clock(d)` | skew the clock observed through `engine.Now(ctx)` by `d`, to drive time-dependent logic (token/lease expiry, backoff) |
| `SlowReader(rate)` / `SlowWriter(rate)` | throttle an `adapter/io`-wrapped stream to `rate` bytes/sec; `rate == 0` blocks until the context is done (a stream that never ends) |
| `Truncate(n)` | cut an `adapter/io`-wrapped stream off after `n` bytes — a reader returns `io.EOF`, a writer `io.ErrShortWrite` |

Attach one with `engine.WithFault(f)` or several (executed in order, short-circuiting on the first error) with `engine.WithFaults(f1, f2, …)`.

`Clock` only moves the time returned by `engine.Now(ctx)`, not the OS clock — read `engine.Now(ctx)` instead of `time.Now()` in the code whose clock-dependent behavior you want to test.

The stream-shaping faults (`SlowReader`/`SlowWriter`/`Truncate`) only take effect through the `adapter/io` reader/writer wrappers; on other adapters they are inert.

## Selecting which operations to hit

Matchers narrow a rule to specific operations. A rule with no matchers matches everything.

| Matcher | Matches |
|---------|---------|
| `MatchKind(kinds…)` | ops from the given adapter kinds |
| `MatchName(glob)` | ops whose name satisfies `path.Match` (`*`, `?`, `[…]`; `*` does not cross `/`) |
| `MatchAttr(key, value)` | ops whose `Attrs[key] == value` |
| `MatchPredicate(fn)` | ops for which `fn(ctx, op)` returns true |

## Controlling how often a rule fires

Counters decide which *matched* operations actually fire the faults.

| Counter | Fires on |
|---------|----------|
| `Always()` *(default)* | every match |
| `Times(n)` | the first `n` matches |
| `Range(from, to)` | matches `[from:to]`, 1-indexed inclusive |
| `Probability(p, seed)` | each match independently with probability `p` (seeded for reproducibility) |
| `Sequence(fire []bool)` | the matches whose index is `true`, in order (used by golden replay) |

## Production safety rails

Pass these as `engine.New(...)` options to bound the blast radius when running chaos against a live system:

- `WithFailureBudget(maxErrorRate, window)` — stop injecting once the observed error rate over a sliding window reaches the threshold, so chaos can't take a dependency fully down.
- `WithRateLimit(rps)` — cap the number of faults that fire per second (global across rules).
- `WithMaxConcurrent(n)` — cap simultaneously in-flight faulted calls.
- `WithProductionGuard(check)` — make `New` **panic** if `check()` returns true (e.g. an env var that marks a forbidden environment).
- `WithKillSwitch(fn)` — suppress all faults whenever `fn(ctx, op)` returns true.
- `Disable()` / `Enable()` — flip an atomic kill switch at runtime; `Reset()` clears all rules and counters.

A worked example wiring all of these together lives in [`examples/prod-safety-rails`](examples/prod-safety-rails/).

### Linting rules

`engine.Lint(rules)` and `engine.LintSpecs(specs)` flag blast-radius hazards — an unconstrained rule that injects a panic or connection drop, a `"*"` name glob, a probability `>= 1.0`, latency beyond a 5s ceiling, or two specs overlapping on the same kind+glob. Gate a build on `report.OK()` (no high-severity findings).

The [`cmd/chaotic-points`](cmd/chaotic-points/) CLI is the static-analysis companion: it discovers `chaos.Point` / `chaos.PointWith` call sites in a module and gates a rules config against typo'd explicit-point names, so a rule targeting `checkout.afterCommt` is caught before it silently never fires.

```bash
go run github.com/RomanAgaltsev/chaotic/cmd/chaotic-points lint --rules chaos.json ./...
```

## Config-driven rules (no recompile)

Rules can be loaded from declarative config instead of code, so you can change chaos behavior on a running service. The serializable form is `engine.RuleSpec`; `engine.BuildRule` validates and converts it.

### From a YAML file (with live reload)

```yaml
# chaos.yaml
meta:
  version: 1
rules:
  - name: flaky-users-api
    kinds: [http_client]
    name_glob: /users/*
    counter: { type: times, n: 3 }
    faults:
      - { type: latency, duration: 250ms }
      - { type: error, message: "upstream timeout" }
```

```go
rs, err := file.Load("chaos.yaml")          // one-shot
eng := engine.New(engine.WithRuleSource(rs))

// or watch the file and hot-reload on change:
go file.Watch(ctx, "chaos.yaml", eng, logger)
```

Config is an untrusted trust boundary, so `BuildRule` rejects absurd values (e.g. latency above a 5-minute cap) instead of honoring them silently.

### From an admin HTTP endpoint

The handler routes with an internal `http.ServeMux` rooted at `/`, so mount it behind `http.StripPrefix`:

```go
// import srchttp "github.com/RomanAgaltsev/chaotic/source/http"
mux.Handle("/chaos/", http.StripPrefix("/chaos", srchttp.New(eng,
    srchttp.WithWritable(true),                      // allow the mutating routes below
    srchttp.WithAuth(func(tok string) bool { ... }), // bearer-token gate
)))
```

| Route | Effect |
|-------|--------|
| `GET /` | return the whole rule set as a YAML document |
| `POST` / `PUT /` | replace the whole rule set (writable) |
| `GET /rules` | list rule names with their hit counts |
| `GET /rules/{name}/count` | return one rule's hit count |
| `PUT /rules/{name}` | install a single rule from a `source/terms` string in the body (writable) |
| `DELETE /rules/{name}` | remove one rule (writable) |

Mutating routes are rejected unless `WithWritable(true)` is set. Rules are swapped atomically via `Engine.ReplaceRules`, so in-flight evaluations never see a torn set.

### From a one-line terms string

`source/terms` parses a compact single-line DSL into rules — handy for an env var, a CLI flag, or a quick test. `terms.Parse` yields `[]engine.RuleSpec`; `terms.Compile` yields ready-to-add `[]engine.Rule`.

```go
rules, err := terms.Compile(`flaky: kind(http_client),name(/users/*)=2*latency(200ms)`)
eng := engine.New()
for _, r := range rules {
    eng.AddRule(r)
}
```

A clause reads `name: <matchers>=<counter>*<faults>` — e.g. `2*error("payment down")` is `Times(2)` of `fault.Error`. See [`examples/terms-dsl`](examples/terms-dsl/).

### From an environment variable

`source/env` reads a `source/terms` string from an environment variable, so an already-built binary can be faulted at process start with no recompile:

```go
rs, err := env.FromEnv("CHAOTIC_RULES") // "" defaults to CHAOTIC_RULES
eng := engine.New(engine.WithRuleSource(rs))
```

It never reads the environment in `init()` — the caller decides when and whether, keeping production opt-in. An unset or empty variable yields an empty rule set, so the engine stays a no-op. Pairs naturally with `engine.WithProductionGuard`.

## Observers

Attach an `Observer` with `engine.WithObserver` to receive an event each time a named rule fires or is skipped:

```go
type Observer interface {
    RuleFired(ruleName string, op Op, action Action)
    RuleSkipped(ruleName string, op Op, reason string)
}
```

An observer may *additionally* implement `RichObserver` to receive per-fault detail (`FaultInjected`) the base interface can't carry — emitted after a non-short-circuiting fault's `Apply` returns (latency, jitter). Faults that short-circuit the call (error, panic, connection drop) do not produce a `FaultEvent`.

Ready-made observers live in their own modules so the core stays dependency-free:

- **`observer/slog`** — structured logs of fires and skips.
- **`observer/prometheus`** — `chaotic_rule_fires_total{rule}`, `chaotic_rule_skips_total{rule,reason}`, and the `chaotic_fault_latency_seconds{rule}` histogram (a `RichObserver`). Label values are truncated to 64 characters to bound series cardinality.
- **`observer/otel`** — OpenTelemetry fire/skip counters, plus (as a `RichObserver`) a `chaotic.fault_injected` event on the active span.

## Testing helpers

The `chaostest` package and its subpackages integrate with `testing.TB`:

- **`chaostest.New(t, opts…)`** — a fresh engine bound to `t.Cleanup`, so faults never leak between tests (safe with `t.Parallel`).
- **`chaostest.AssertHits(t, eng, name, want)`** / **`AssertEventsExhausted(t, eng)`** — assert a named rule fired exactly *N* times, or that every configured rule fired at least once.
- **`chaostest/quick`** — one-liners for the common setups: `FailFirst`, `SlowAlways`, `PanicOnce`.
- **`chaostest/golden`** — record a fault fire-sequence from one run (`go test -chaos-update-golden`) and `Replay` it deterministically to turn a flaky CI failure into a reproducible local one.
- **`chaostest/scenarios`** — one-call recipes for common failure modes: `DatabaseOutageCascade`, `ThunderingHerdAfterDeploy`, `SlowLeaderElection`, `PartialNetworkPartition`.
- **`chaostest/property`** — a property-testing harness. `property.Test(t, gens, body, opts…)` runs your invariant against many randomized rule sets and, on failure, delta-debugs the input down to the single culprit generator.
- **`chaostest/bench`** — `bench.Run(b, eng, profiles, body)` runs one benchmark body across a series of named chaos profiles so you can measure overhead under each.

## Compiling chaos out

Build with the `chaos_off` tag and every adapter wrapper, `chaos.Point`, and `chaos.WithEngine` compile down to a bare passthrough — no engine, no allocations, nothing to strip from a production binary by hand:

```bash
go build -tags chaos_off ./...
```

## Examples

Runnable, tested scenarios live in [`examples/`](examples/). Each has a `main.go` (`go run .`), a `main_test.go`, and a `README.md`.

| Scenario | Demonstrates | Adapter |
|----------|--------------|---------|
| [retry-http](examples/retry-http/) | a retry loop recovers from a transient injected failure | `adapter/http` |
| [circuit-breaker](examples/circuit-breaker/) | a breaker opens after repeated injected failures | `adapter/http` |
| [db-conn-pool](examples/db-conn-pool/) | the pool evicts a poisoned conn (`ConnDrop` → `ErrBadConn`) | `adapter/sql` |
| [grpc-stream-reconnect](examples/grpc-stream-reconnect/) | a stream client reconnects after an injected `Unavailable` | `adapter/grpc` |
| [pgx-pool](examples/pgx-pool/) | pool-level chaos on a pgx pool (integration-gated) | `adapter/pgx` |
| [redis-cache-fallback](examples/redis-cache-fallback/) | a read-through cache falls back to the DB when Redis fails | `adapter/redis` |
| [kafka-write-retry](examples/kafka-write-retry/) | a producer retries through a transient Kafka write outage (needs Docker) | `adapter/kafka` |
| [nats-request-retry](examples/nats-request-retry/) | a request/reply caller retries through a transient NATS outage | `adapter/nats` |
| [mongo-read-fallback](examples/mongo-read-fallback/) | a read retries through a transient MongoDB step-down (needs Docker) | `adapter/mongo` |
| [rabbitmq-publish-retry](examples/rabbitmq-publish-retry/) | a publisher retries through a transient RabbitMQ outage (needs Docker) | `adapter/rabbitmq` |
| [aws-dynamodb-retry](examples/aws-dynamodb-retry/) | the AWS SDK's own retryer recovers from an injected outage | `adapter/aws` |
| [net-conn-drop](examples/net-conn-drop/) | a read loop retries through a transient connection drop | `adapter/net` |
| [slow-body-read](examples/slow-body-read/) | a `Truncate` fault cuts a response body mid-JSON; the reader surfaces a clean parse error | `adapter/io` |
| [chaos-point](examples/chaos-point/) | an explicit `chaos.Point` guards a post-commit hook | `chaos` |
| [clock-skew](examples/clock-skew/) | a token expires once `fault.Clock` skews `engine.Now` past its TTL | `fault.Clock` |
| [terms-dsl](examples/terms-dsl/) | a one-line terms string activates chaos with no rule-building code | `source/terms` |
| [prod-safety-rails](examples/prod-safety-rails/) | failure budget + caps + guard + kill switch bound the blast radius | `engine` |
| [observability-during-chaos](examples/observability-during-chaos/) | an `engine.Observer` records a chaos fire for logs/metrics/traces | `engine` |

Per-symbol godoc examples live next to each package on [pkg.go.dev](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic).

## Repository layout

chaotic is a Go workspace (`go.work`) of several modules so consumers pull in only the dependencies they use:

| Module | Contents | Third-party deps |
|--------|----------|------------------|
| `github.com/RomanAgaltsev/chaotic` | engine, faults, explicit points, terms DSL, `env` rule source, test helpers (`chaostest` + `quick`/`golden`/`scenarios`/`property`/`bench`), HTTP / HTTP-server / SQL adapters | none (stdlib) |
| `…/adapter/grpc` | gRPC interceptors | `google.golang.org/grpc` |
| `…/adapter/pgx` | native pgx v5 pool & conn wrappers | `github.com/jackc/pgx/v5` |
| `…/adapter/redis` | go-redis v9 hook | `github.com/redis/go-redis/v9` |
| `…/adapter/kafka` | segmentio kafka-go writer & reader | `github.com/segmentio/kafka-go` |
| `…/adapter/nats` | nats.go connection wrapper | `github.com/nats-io/nats.go` |
| `…/adapter/mongo` | mongo-driver v2 collection/db/client | `go.mongodb.org/mongo-driver/v2` |
| `…/adapter/rabbitmq` | amqp091-go channel & connection | `github.com/rabbitmq/amqp091-go` |
| `…/adapter/aws` | aws-sdk-go-v2 middleware | `github.com/aws/aws-sdk-go-v2` |
| `…/adapter/net` | raw `net.Conn` / `Listener` / `Dialer` | none (stdlib) |
| `…/adapter/io` | `io.Reader` / `io.Writer` stream wrappers | none (stdlib) |
| `…/observer/slog` · `/prometheus` · `/otel` | ready-made observers | respective backends |
| `…/source/file` · `/http` | YAML rule files & admin endpoint | `gopkg.in/yaml.v3` |
| `…/cmd/chaotic-points` | static-analysis CLI: discover points, gate rule configs | `golang.org/x/tools` |

Run the tests:

```bash
go test ./...                  # core module
go -C adapter/grpc  test ./... # gRPC submodule
go -C adapter/pgx   test ./... # pgx submodule (integration tests are build-gated)
go -C adapter/redis test ./... # any other submodule follows the same pattern
```

Every module under `adapter/`, `observer/`, `source/file`, `source/http`, and `cmd/` is its own Go module — run its tests with `go -C <dir> test ./...` (CI discovers them all automatically).

The repo uses [Taskfile](Taskfile.yml) for common workflows — see `task --list`.

## Contributing

Issues and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow and [SECURITY.md](SECURITY.md) to report a vulnerability.

## License

[MIT](LICENSE) © Roman Agaltsev
