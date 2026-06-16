// Command observability-during-chaos shows an engine.Observer capturing a chaos
// fire: when a rule injects a fault on an HTTP call, the observer records the
// RuleFired event — exactly where a real app forwards it to logs, metrics, or
// traces (see observer/slog, observer/prometheus, observer/otel).
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	chaoshttp "github.com/RomanAgaltsev/chaotic/adapter/http"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// recorder is a minimal engine.Observer that records the names of fired rules.
type recorder struct {
	mu    sync.Mutex
	fired []string
}

func (r *recorder) RuleFired(name string, _ engine.Op, _ engine.Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fired = append(r.fired, name)
}

func (r *recorder) RuleSkipped(_ string, _ engine.Op, _ string) {}

func (r *recorder) Fired() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.fired...)
}

// run wires obs into an engine, faults one HTTP call, and makes that call so the
// observer fires. It returns the (injected) error from the call.
func run(obs engine.Observer) error {
	eng := engine.New(engine.WithObserver(obs)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(errors.New("injected"))),
	).Named("http-fail"))

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	return err
}

func main() {
	rec := &recorder{}
	_ = run(rec)
	fmt.Fprintf(os.Stdout, "observer saw fires: %v\n", rec.Fired())
}
