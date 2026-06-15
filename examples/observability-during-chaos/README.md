# observability-during-chaos

Demonstrates that an [`engine.Observer`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/engine#Observer)
sees every chaos fire. A rule injects an error on an HTTP client call; the
attached observer records the `RuleFired` event for rule `http-fail` — the hook
point where a real service forwards the fire to logs, metrics, or traces.

```bash
go run .        # prints: observer saw fires: [http-fail]
go test ./...   # asserts the observer recorded the fire
```
