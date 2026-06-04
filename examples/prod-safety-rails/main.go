// Command prod-safety-rails shows the production bounds that keep chaos from
// becoming the outage: a failure budget, a max-concurrent cap, a production
// guard, and a kill switch.
// Run with `go run .`.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// newEngine wires every production bound. The failure budget stops injection
// once the observed error rate over the window reaches 50%, so chaos cannot
// drive the dependency fully down. The guard panics New if CHAOS_FORBIDDEN=1;
// the kill switch suppresses all faults if CHAOS_KILL=1.
func newEngine() *engine.Engine {
	return engine.New(
		engine.WithFailureBudget(0.5, 10),
		engine.WithMaxConcurrent(5),
		engine.WithProductionGuard(func() bool { return os.Getenv("CHAOS_FORBIDDEN") == "1" }),
		engine.WithKillSwitch(func(context.Context, engine.Op) bool { return os.Getenv("CHAOS_KILL") == "1" }),
	).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(errors.New("injected"))),
	).Named("guarded"))
}

// run makes total calls through the bounded engine and reports how many were
// actually faulted. The failure budget ensures this is greater than zero but
// well below total: chaos backs off instead of taking the dependency down.
func run() (injected, total int) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, newEngine())}
	const n = 50
	for range n {
		resp, err := client.Get(srv.URL)
		if err != nil {
			injected++
			continue
		}
		_ = resp.Body.Close()
	}
	return injected, n
}

func main() {
	injected, total := run()
	fmt.Printf("%d/%d calls were faulted; the failure budget stopped the rest\n", injected, total)
}
