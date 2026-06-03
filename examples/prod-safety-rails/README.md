# prod-safety-rails

The production bounds that stop chaos from becoming the outage.

`newEngine` wraps an always-fire error rule with:

- `WithFailureBudget(0.5, 10)` — stop injecting once the error rate over the
  last 10 calls reaches 50%, so chaos backs off instead of taking the
  dependency fully down;
- `WithMaxConcurrent(5)` — cap simultaneously faulted calls;
- `WithProductionGuard(...)` — `New` panics if `CHAOS_FORBIDDEN=1`;
- `WithKillSwitch(...)` — all faults suppressed if `CHAOS_KILL=1`.

`run` makes 50 calls; the test asserts that some are faulted but not all — the
budget engaged. Try `CHAOS_KILL=1 go run .` to see chaos fully suppressed.