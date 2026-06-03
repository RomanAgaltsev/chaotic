# chaotic
[![test](https://github.com/ag4r/chaotic/actions/workflows/test.yml/badge.svg)](https://github.com/ag4r/chaotic/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/ag4r/chaotic/branch/main/graph/badge.svg)](https://codecov.io/gh/ag4r/chaotic)
[![Go Reference](https://pkg.go.dev/badge/github.com/ag4r/chaotic.svg)](https://pkg.go.dev/github.com/ag4r/chaotic)
[![Go Report Card](https://goreportcard.com/badge/github.com/ag4r/chaotic)](https://goreportcard.com/report/github.com/ag4r/chaotic)
[![Release](https://img.shields.io/github/v/release/ag4r/chaotic)](https://github.com/ag4r/chaotic/releases)
[![License: MIT](https://img.shields.io/github/license/ag4r/chaotic)](./LICENSE)

In-process Go chaos library. Wrap your integration boundaries — `http.RoundTripper`, `net/http` middleware, `database/sql` driver, gRPC interceptors — and inject latency, errors, panics, and connection drops on demand. When no rules are configured the wrappers are near-zero-cost passthroughs; chaotic is safe to leave linked everywhere.

## Status

v1, test-only mode. Production-mode hooks (HTTP admin endpoint, file-watcher rule source, observers) are forward-compatible additions planned for v2.

## Install

```bash
go get github.com/ag4r/chaotic
go get github.com/ag4r/chaotic/adapter/grpc   # only if you need gRPC chaos
```

## Quick example

```go
package main

import (
	"io"
	"net/http"
	"time"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
	"github.com/ag4r/chaotic/chaostest"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestRetriesOnTransientError(t *testing.T) {
	eng := chaostest.New(t)
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.MatchName("/users/*"),
		engine.Times(2),
		engine.WithFaults(fault.Latency(200*time.Millisecond), fault.Error(io.ErrUnexpectedEOF)),
	).Named("transient-failure"))

	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
	// ... call code under test that uses `client` and is supposed to retry ...

	chaostest.AssertHits(t, eng, "transient-failure", 2)
	chaostest.AssertEventsExhausted(t, eng)
}
```

## Modules

This repo contains two Go modules:

- `github.com/ag4r/chaotic` — engine, faults, and the HTTP / HTTP server / SQL adapters. Stdlib only.
- `github.com/ag4r/chaotic/adapter/grpc` — gRPC interceptors. Depends on `google.golang.org/grpc`.

A `go.work` file at the repo root makes the workspace resolve both modules during development. Run tests with:

```bash
go test ./...                 # main module
go -C adapter/grpc test ./... # gRPC submodule
```

## Observers

Attach an `Observer` with `engine.WithObserver` to receive an event each time a
named rule fires or is skipped:

```go
type Observer interface {
	RuleFired(ruleName string, op Op, action Action)
	RuleSkipped(ruleName string, op Op, reason string)
}
```

An observer may *additionally* implement `RichObserver` to receive per-fault
detail the base interface cannot carry. The engine type-asserts for it once, at
construction, and calls `FaultInjected` after a fault's `Apply` returns without
error (latency and jittered faults, plus any custom no-op fault). Faults that
short-circuit the call — error, panic, connection drop — do **not** produce a
`FaultEvent`.

```go
type RichObserver interface {
	Observer
	FaultInjected(ctx context.Context, ev FaultEvent)
}

type FaultEvent struct {
	Rule      string
	Op        Op
	FaultKind fault.Kind
	Latency   time.Duration // exact for Latency; the max bound for Jittered; 0 otherwise
}
```

Ready-made observers live in their own submodules so the core stays
dependency-free:

- `observer/slog` — structured logs of fires and skips.
- `observer/prometheus` — `chaotic_rule_fires_total{rule}`,
  `chaotic_rule_skips_total{rule,reason}`, and the
  `chaotic_fault_latency_seconds{rule}` histogram (a `RichObserver`). Label
  values are truncated to 64 characters to bound series cardinality.
- `observer/otel` — OpenTelemetry fire/skip counters, plus (as a `RichObserver`)
  a `chaotic.fault_injected` event on the span active in the call's context.

### `Op.Attrs` key conventions

`Op.Attrs` is a `map[string]string`. Adapters populate it with stable,
low-cardinality keys so observer labels stay consistent. Match on these with
`engine.MatchAttr`:

| Adapter        | `Op.Kind`        | `Op.Name`         | `Op.Method`        | `Op.Attrs` keys |
|----------------|------------------|-------------------|--------------------|-----------------|
| HTTP client    | `OpHTTPClient`   | request path      | HTTP method        | `host`          |
| HTTP server    | `OpHTTPServer`   | request path      | HTTP method        | `remote`        |
| SQL            | `OpSQL`          | statement class   | statement class    | `query`         |
| gRPC client    | `OpGRPCClient`   | full method       | `unary` / `stream` | —               |
| gRPC server    | `OpGRPCServer`   | full method       | `unary` / `stream` | —               |

Keep custom attribute values low-cardinality: they become metric labels and
unbounded values (raw SQL text, user IDs) blow up an observer's memory.