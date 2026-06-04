//go:build !chaos_off

package http_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleWrapTransport() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "ok")
	}))
	defer srv.Close()

	// Fail only the first request with a transient error.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("transient"))),
	).Named("flap"))

	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}

	resp1, err := client.Get(srv.URL)
	if resp1 != nil {
		resp1.Body.Close()
	}
	fmt.Println("attempt 1 failed:", err != nil)

	resp, err := client.Get(srv.URL)
	if err != nil {
		fmt.Println("attempt 2 error:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("attempt 2 status:", resp.StatusCode)
	// Output:
	// attempt 1 failed: true
	// attempt 2 status: 200
}

func ExampleWrapTransport_httpStatus() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.HTTPStatus(503)),
	).Named("degrade"))
	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}

	// The client sees a real 503 response (not a transport error) to test
	// retry/handling code.
	resp, err := client.Get(srv.URL + "/")
	fmt.Println("err:", err)
	fmt.Println("status:", resp.StatusCode)
	_ = resp.Body.Close()
	// Output:
	// err: <nil>
	// status: 503
}
