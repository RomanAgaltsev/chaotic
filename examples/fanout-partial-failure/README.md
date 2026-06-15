# fanout-partial-failure

Demonstrates graceful degradation under partial failure. `FanOut` queries three
backends concurrently; a chaos rule scoped to path `/b`
([`MatchName`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/engine#MatchName))
faults that one branch. The aggregator returns the two branches that succeeded
(`/a`, `/c`) instead of failing the whole request.

```bash
go run .        # prints: succeeded: [/a /c]
go test ./...   # asserts the faulted branch is dropped, and the clean path returns all three
```
