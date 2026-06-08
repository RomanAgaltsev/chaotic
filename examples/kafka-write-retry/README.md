# Kafka write-with-retry under chaos

A producer that retries through a transient Kafka outage. This example proves the
retry path actually works by using the chaotic `adapter/kafka` wrapper to make
writes fail on demand — no real outage, no flaky timing.

## What it shows

- Wrap a live writer so its writes are chaos-aware:
  `cw := chaoskafka.WrapWriter(w, eng)`.
- A rule that `ConnDrop`s the first two writes (`Times(2)`) simulates a transient
  broker outage. `ConnDrop` maps to `io.ErrUnexpectedEOF` and never touches the
  real writer, so the retry's next attempt lands on the still-open writer.
- `WriteWithRetry` keeps trying and the message is delivered; the rule's hit
  counter proves the outage was exercised exactly twice.

## Run it

```
go test ./...
```

The test uses [testcontainers-go](https://golang.testcontainers.org/) to start a
real Kafka broker, so **Docker must be running**. Without Docker the test skips
(it does not fail).