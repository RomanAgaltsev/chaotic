# chaotic

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