// Command retry-http demonstrates a retry loop recovering from a transient
// fault injected into an http.Client's transport. Run with `go run .`.
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

// newEngine fails only the first HTTP client call, then becomes inert.
func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("transient network error"))),
	).Named("http-flap"))
}

// newServer is a backend that always succeeds.
func newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "ok")
	}))
}

// getWithRetry retries the GET up to attempts times.
func getWithRetry(client *http.Client, url string, attempts int) (*http.Response, error) {
	var err error
	for range attempts {
		var req *http.Request
		if req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil); err != nil {
			return nil, err
		}
		var resp *http.Response
		if resp, err = client.Do(req); err == nil {
			return resp, nil
		}
	}
	return nil, err
}

func run() error {
	srv := newServer()
	defer srv.Close()
	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, newEngine())}
	resp, err := getWithRetry(client, srv.URL, 3)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	fmt.Fprintln(os.Stdout, "succeeded after retry, status", resp.StatusCode)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "FAILED:", err)
	}
}
