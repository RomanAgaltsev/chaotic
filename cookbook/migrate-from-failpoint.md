# Migrating from pingcap/failpoint

`pingcap/failpoint` rewrites your source at build time and substitutes return
**values** at labeled points. chaotic instead wraps **boundaries** (HTTP, SQL,
gRPC, …) or explicit [`chaos.Point`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/chaos)
call sites and injects **faults** (latency, error, panic, conn-drop) — no build
step, no codegen.

## Pattern mapping

| pingcap/failpoint | chaotic |
|-------------------|---------|
| `failpoint.Inject("label", func(v failpoint.Value) { … })` | `chaos.Point(ctx, "label")` at the same site; a rule on `kind(explicit),name(label)` decides what fires |
| `failpoint.Enable("path/label", "return(42)")` | a rule on the matching boundary/point, or `chaostest.Enable(t, eng, "kind(explicit),name(label)=error(\"boom\")")` |
| `failpoint.Disable("path/label")` | not needed — `chaostest.Enable` registers `t.Cleanup`; or call `eng.Reset()` |
| `GO_FAILPOINTS=...` env activation | [`source/env`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/source/env) parsed into a `RuleSet`, paired with `WithProductionGuard` |
| comment-directive codegen (`// +failpoint`) | none — chaotic has no build step; place a `chaos.Point` call instead |

## What does not map: value substitution

failpoint's signature feature, `return(value)`, makes a function return a chosen
value. chaotic deliberately does **not** do this — it injects faults, not mock
return values (see the roadmap's "faults, not mocks"). To port a
`return(badValue)` failpoint, restructure the assertion around the **fault** the
bad value would have caused: inject the error/latency/panic and assert your code
handles it. For genuine return-value mocking, use `gomock` or a hand-rolled fake
alongside chaotic.

## Why the difference

chaotic's boundary-wrapping model means chaos is configured the same way in tests
and (guarded) in production, with no separate build of the binary, and every
fault is bounded by the engine's safety rails (kill switch, rate limit,
concurrency cap, failure budget).
