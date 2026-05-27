package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"syscall"
	"testing"
	"time"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func newEngine(t *testing.T, r engine.Rule) *engine.Engine {
	t.Helper()
	e := engine.New().AddRule(r)
	t.Cleanup(e.Reset)
	return e
}

func TestNoOpWhenEngineNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client := &http.Client{
		Transport: chaoshttp.WrapTransport(http.DefaultTransport, nil),
	}
	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestErrorFaultWrappedAsUrlError(t *testing.T) {
	target := errors.New("boom")
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Error(target)),
	))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	rt := chaoshttp.WrapTransport(http.DefaultTransport, e)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rt.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("err = %T %v, want *url.Error", err, err)
	}
	if urlErr.Op != "chaos" {
		t.Fatalf("urlErr.Op = %q, want \"chaos\"", urlErr.Op)
	}
	if !errors.Is(err, target) {
		t.Fatalf("underlying error not target: %v", err)
	}
}

func TestLatencyFaultDelaysCall(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.Latency(40*time.Millisecond)),
	))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	client := &http.Client{
		Transport: chaoshttp.WrapTransport(http.DefaultTransport, e),
	}
	start := time.Now()
	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if time.Since(start) < 30*time.Millisecond {
		t.Fatal("call returned too quickly")
	}
}

func TestConnDropFaultReturnsNetOpError(t *testing.T) {
	e := newEngine(t, engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(fault.ConnDrop()),
	))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	client := &http.Client{
		Transport: chaoshttp.WrapTransport(http.DefaultTransport, e),
	}
	resp, err := client.Get(srv.URL + "/")
	if resp != nil {
		resp.Body.Close()
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("err = %T %v, want *net.OpError", err, err)
	}
	if opErr.Op != "dial" {
		t.Fatalf("OpError.Op = %q, want \"dial\"", opErr.Op)
	}
	if !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("err not ECONNRESET: %v", err)
	}
}

func TestOpAttrsIncludeHost(t *testing.T) {
	var gotAttrs map[string]string
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchPredicate(func(_ context.Context, op engine.Op) bool {
			gotAttrs = op.Attrs
			return false
		}),
	))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	client := &http.Client{
		Transport: chaoshttp.WrapTransport(http.DefaultTransport, e),
	}
	resp, err := client.Get(srv.URL + "/x")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotAttrs["host"] == "" {
		t.Fatalf("Attrs.host empty, want set; got %v", gotAttrs)
	}
}
