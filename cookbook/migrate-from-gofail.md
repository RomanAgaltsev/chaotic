# Migrating from etcd-io/gofail

`etcd-io/gofail` codegens failpoints from comments and activates them with a
terms-like string (`gofail.Enable("fp", 'sleep(10)')`). chaotic borrows gofail's
**activation ergonomics** — a compact terms string — but keeps its own model:
boundary/point wrapping and fault injection, no codegen, no value substitution.

## Pattern mapping

| etcd-io/gofail | chaotic |
|----------------|---------|
| `gofail.Enable("fp", 'sleep(200)')` + `defer gofail.Disable("fp")` | `chaostest.Enable(t, eng, "kind(explicit),name(fp)=latency(200ms)")` — auto-cleanup, no `Disable` |
| terms `sleep(d)` | `latency(d)` |
| terms `panic` | `panic("msg")` |
| terms `return("x")` | not supported — faults, not values (see [migrate-from-failpoint](migrate-from-failpoint.md)) |
| `GOFAIL_FAILPOINTS=...` env | [`source/env`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/source/env) — an explicit constructor returning a caller-owned `RuleSet`, never an `init()`-read global |
| codegen-rewritten failpoint label | a [`chaos.Point(ctx, "label")`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/chaos) call (no build step) |

## The one-liner

gofail's ergonomic high point is its in-test one-liner. chaotic's equivalent:

```go
func TestRetry(t *testing.T) {
    eng := chaostest.New(t)
    names := chaostest.Enable(t, eng, `kind(http_client),name(/users/*)=2*latency(200ms)->error("boom")`)
    client := &http.Client{Transport: chaoshttp.WrapTransport(nil, eng)}
    // ... exercise the retry loop ...
    chaostest.AssertHits(t, eng, names[0], 2)
}
```

Enable parses the terms grammar, adds the rules, and resets them via t.Cleanup when the test ends.

## What does not map
gofail’s codegen and value substitution are intentionally not adopted: chaotic wraps boundaries (no build step) and injects faults (not mock return values). See migrate-from-failpoint for the value-substitution porting advice.
