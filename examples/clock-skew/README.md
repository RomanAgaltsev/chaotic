# clock-skew

Demonstrates [`fault.Clock`](https://pkg.go.dev/github.com/ag4r/chaotic/fault#Clock):
a skewed wall clock observed through `engine.Now`.

A token-expiry check reads the clock with `engine.Now(ctx)` instead of
`time.Now()`. A chaos rule attaches a `fault.Clock(2h)` fault to the
`token.validate` explicit point. Before the point fires the token is valid;
after it fires the clock jumps two hours ahead — past the token's one-hour TTL —
and the same token reads as expired. This is the bug class `fault.Clock` exists
to surface: deadline / expiry / timezone arithmetic that only breaks when the
clock is wrong.

go run . go test ./…
