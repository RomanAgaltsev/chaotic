//go:build !chaos_off

package httpsrv_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/ag4r/chaotic/adapter/httpsrv"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleMiddleware() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("overloaded"))),
	).Named("inbound"))

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "handled")
	})
	h = httpsrv.Middleware(eng)(h)

	srv := httptest.NewServer(h)
	defer srv.Close()

	get := func() int {
		resp, err := http.Get(srv.URL)
		if err != nil {
			return -1
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}

	fmt.Println("request 1 status:", get()) // chaos fault -> 500 before the handler
	fmt.Println("request 2 status:", get()) // chaos exhausted -> handler runs
	// Output:
	// request 1 status: 500
	// request 2 status: 200
}

func ExampleMiddleware_httpStatus() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.HTTPStatus(503, "overloaded")),
	).Named("degrade"))

	h := httpsrv.Middleware(eng)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "handler ran")
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/")
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	fmt.Print(resp.StatusCode, " ", string(body))
	// Output: 503 overloaded
}
