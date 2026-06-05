# Redis cache fallback under chaos

Cookbook recipe #10: *Test that Redis cache fallback to DB works.*

A read-through cache (`Store`) reads from Redis and, on any Redis error, falls
back to a backing database. This example proves the fallback path actually works
by using the chaotic `adapter/redis` hook to make Redis fail on demand — no real
outage, no flaky timing.

## What it shows

- Install chaos on a real `*redis.Client` with one line:
  `rc.AddHook(chaosredis.NewHook(eng))`.
- A rule that `ConnDrop`s every `GET` simulates Redis being unreachable.
- The service still returns correct values from the database fallback, and the
  rule's hit counter proves the outage was exercised.

## Run it

go test ./...

The test (`main_test.go`) seeds a database map, drops every Redis `GET`, and
asserts the value is still served from the fallback.