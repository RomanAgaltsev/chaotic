// Command circuit-breaker shows a circuit breaker opening after chaos injects
// repeated failures, after which calls short-circuit instead of calling the
// failing dependency. Run with `go run .`.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	chaoshttp "github.com/RomanAgaltsev/chaotic/adapter/http"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

var errBreakerOpen = errors.New("circuit breaker open")

type breaker struct {
	threshold int
	failures  int
	open      bool
}

func (b *breaker) call(fn func() error) error {
	if b.open {
		return errBreakerOpen
	}
	if err := fn(); err != nil {
		b.failures++
		if b.failures >= b.threshold {
			b.open = true
		}
		return err
	}
	b.failures = 0
	return nil
}

// newEngine fails every HTTP client call so the breaker is guaranteed to open.
func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(errors.New("dependency down"))),
	).Named("always-fail"))
}

// drive makes n calls through a breaker with threshold 3 and returns how many
// of them actually reached the HTTP client (the rest were short-circuited once
// the breaker opened). The count is the engine's hit count for the rule.
func drive(n int) (clientCalls int) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	eng := newEngine()
	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
	b := &breaker{threshold: 3}
	for range n {
		_ = b.call(func() error {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			return resp.Body.Close()
		})
	}
	return eng.Hits("always-fail")
}

func main() {
	calls := drive(10)
	fmt.Fprintf(os.Stdout, "10 requests, breaker threshold 3: dependency was called %d times\n", calls)
}
