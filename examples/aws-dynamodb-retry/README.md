# AWS SDK retry under chaos (DynamoDB, no account, no Docker)

This example proves the AWS SDK for Go v2's **own** retryer recovers from a
transient outage injected by the chaotic `adapter/aws` middleware — with no AWS
account and no Docker. A local `httptest` server stands in for DynamoDB.

## What it shows

- Install chaos on an `aws.Config` with one line:
  `chaosaws.AppendChaosMiddleware(&cfg, eng)`. Every client built from that config
  is chaos-aware.
- Because the middleware sits at the **Finalize** step (after retry setup), an
  injected error is classified and retried by the SDK exactly like a real failure.
- A rule that `ConnDrop`s the first two attempts (`Times(2)`) is fully absorbed by
  the SDK's default retryer (max 3 attempts); the request still succeeds, and the
  rule's hit counter proves the outage was exercised twice.

## Run it

go test ./...

No Docker, no credentials, no network egress — the DynamoDB endpoint is a local
`httptest` server.
