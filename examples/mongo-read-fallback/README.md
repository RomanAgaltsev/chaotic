# MongoDB read-with-retry under chaos

A read that retries through a transient MongoDB failover (primary step-down). This
example proves the retry path actually works by using the chaotic `adapter/mongo`
wrapper to make reads fail on demand — no real outage, no flaky timing.

## What it shows

- Wrap a live client so every collection it hands out is chaos-aware:
  `coll := chaosmongo.WrapClient(client, eng).Database("app").Collection("users")`.
- A rule that `ConnDrop`s the first two reads (`Times(2)`) simulates a transient
  failover. `ConnDrop` maps to a retryable `mongo.CommandError` and never touches
  the real collection, so the retry's next read lands on the still-connected client.
- `ReadUserWithRetry` keeps trying and the document is returned; the rule's hit
  counter proves the failover was exercised exactly twice.

## Run it

go test ./...

The test uses [testcontainers-go](https://golang.testcontainers.org/) to start a
real MongoDB, so **Docker must be running**. Without Docker the test skips (it does
not fail).