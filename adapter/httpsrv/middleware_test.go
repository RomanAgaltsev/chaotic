package httpsrv_test

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ag4r/chaotic/adapter/httpsrv"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newEngine(t *testing.T, r engine.Rule) *engine.Engine {
	t.Helper()
	e := engine.New().AddRule(r)
	t.Cleanup(e.Reset)
	return e
}

func TestMiddlewareIsTransparentWhenEngineEmpty(t *testing.T) {
	called := false
	h := httpsrv.Middleware(engine.New())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !called {
		t.Fatal("downstream handler not called")
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatal("downstream handler not called")
	}
}

func TestErrorFaultReturns500AndSkipHandler(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.Error(errors.New("boom"))),
	))
	called := false
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatal("downstream handler not called")
	}
	if called {
		t.Fatal("downstream handler not called")
	}
}

func TestLatencyFaultDelaysBeforeHandler(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.Latency(40*time.Millisecond)),
	))
	var handlerStart time.Time
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerStart = time.Now()
		w.WriteHeader(http.StatusOK)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	reqStart := time.Now()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if delay := handlerStart.Sub(reqStart); delay < 30*time.Millisecond {
		t.Fatalf("handler ran too early, delay was %v", delay)
	}
}

func TestConnDropFaultClosesConnection(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.ConnDrop()),
	))
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	// Use a fresh client with no keep-alive so the connection drop is visible.
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{
		Transport: tr,
		Timeout:   2 * time.Second,
	}
	resp, err := client.Get(srv.URL + "/")
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected error from dropped connection, got nil")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) && !errors.Is(err, io.EOF) {
		// Either a net.Error or an EOF is acceptable evidence of a forced close.
		t.Fatalf("err = %T %v, want net.Error or EOF", err, err)
	}
}
