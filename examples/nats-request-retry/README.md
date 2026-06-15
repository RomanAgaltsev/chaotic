# NATS request-with-retry under chaos

A request/reply caller that retries through a transient NATS outage. This example
proves the retry path actually works by using the chaotic `adapter/nats` wrapper to
make requests fail on demand — no real outage, no flaky timing, and no Docker.

## What it shows

- Wrap a live connection for per-call chaos: `cc := chaosnats.WrapConn(nc, eng)`.
  (For dial/reconnect chaos instead, pass `chaosnats.Option(eng)` to `nats.Connect`.)
- A rule that `ConnDrop`s the first two requests (`Times(2)`) simulates a transient
  outage. `ConnDrop` maps to a transient `nats.ErrConnectionClosed` and never
  touches the real connection, so the retry's next request lands on the still-open
  connection.
- `RequestWithRetry` keeps trying and the reply arrives; the rule's hit counter
  proves the outage was exercised exactly twice.

## Run it

go test ./...

The test runs an **in-process** nats-server (no Docker, no external broker).
