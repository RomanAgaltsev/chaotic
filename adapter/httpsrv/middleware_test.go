//go:build !chaos_off

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

func TestMiddlewareReportsServerErrorToBudget(t *testing.T) {
	eng := engine.New(engine.WithFailureBudget(0.5, 2)).
		AddRule(engine.NewRule(
			engine.MatchKind(engine.OpHTTPServer),
			engine.WithFault(fault.Latency(0)),
		).Named("slow"))
	// Downstream handler always 500s.
	h := httpsrv.Middleware(eng)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	for range 2 {
		reс := httptest.NewRecorder()
		h.ServeHTTP(reс, httptest.NewRequest(http.MethodGet, "/x", nil))
	}
	hits := eng.Hits("slow")
	if hits != 2 {
		t.Fatalf("Hits = %d, want 2", hits)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/y", nil))
	if eng.Hits("slow") != hits {
		t.Fatalf("rule fired despite over-budget: Hits %d -> %d", hits, eng.Hits("slow"))
	}
}

func TestHTTPStatusFaultRendersStatus(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.HTTPStatus(503, "overloaded")),
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
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	if string(body) != "overloaded\n" { // http.Error appends a newline
		t.Fatalf("body = %q, want %q", string(body), "overloaded\n")
	}
	if called {
		t.Fatal("handler ran despite an injected status fault")
	}
}

func TestHTTPStatusFaultDefaultBody(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.HTTPStatus(429)),
	))
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp.StatusCode)
	}
	if string(body) != http.StatusText(http.StatusTooManyRequests)+"\n" {
		t.Fatalf("body = %q, want default status text", string(body))
	}
}

func TestHeaderStripFaultMutatesRequestAndContinues(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.HeaderStrip("X-Trace-Id")),
	))
	var seen string
	called := false
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		seen = r.Header.Get("X-Trace-Id")
		w.WriteHeader(http.StatusOK)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	req.Header.Set("X-Trace-Id", "present")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !called {
		t.Fatal("handler did not run; header fault must continue, not abort")
	}
	if seen != "" {
		t.Fatalf("handler saw X-Trace-Id = %q, want it stripped", seen)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestHeaderInjectFaultMutatesRequest(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPServer),
		engine.WithFault(fault.HeaderInject("X-Injected", "yes")),
	))
	var seen string
	h := httpsrv.Middleware(e)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-Injected")
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if seen != "yes" {
		t.Fatalf("handler saw X-Injected = %q, want %q", seen, "yes")
	}
}
