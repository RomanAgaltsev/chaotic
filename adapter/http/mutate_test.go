//go:build !chaos_off

package http_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	chaoshttp "github.com/RomanAgaltsev/chaotic/adapter/http"
	"github.com/RomanAgaltsev/chaotic/engine"
)

// stubRT returns a fixed 200 response with body, or err if set.
type stubRT struct {
	body string
	err  error
}

func (s stubRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Request:    req,
	}, nil
}

func ruleEngine(f func(*http.Response) *http.Response) *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(chaoshttp.MutateResponse(f)),
	))
}

func TestMutateResponseRewritesStatus(t *testing.T) {
	eng := ruleEngine(func(r *http.Response) *http.Response { r.StatusCode = http.StatusServiceUnavailable; return r })
	rt := chaoshttp.WrapTransport(stubRT{body: "ok"}, eng)
	req, _ := http.NewRequest(http.MethodGet, "http://example.test/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestMutateResponseRewritesBody(t *testing.T) {
	eng := ruleEngine(func(r *http.Response) *http.Response {
		r.Body = io.NopCloser(strings.NewReader(`{"corrupt":true}`))
		return r
	})
	rt := chaoshttp.WrapTransport(stubRT{body: `{"ok":true}`}, eng)
	req, _ := http.NewRequest(http.MethodGet, "http://example.test/x", nil)
	resp, _ := rt.RoundTrip(req)
	defer func() { _ = resp.Body.Close() }()
	got, _ := io.ReadAll(resp.Body)
	if string(got) != `{"corrupt":true}` {
		t.Fatalf("body = %s", got)
	}
}

func TestMutateResponseErrorPathUntouched(t *testing.T) {
	called := false
	eng := ruleEngine(func(r *http.Response) *http.Response { called = true; return r })
	rt := chaoshttp.WrapTransport(stubRT{err: io.ErrUnexpectedEOF}, eng)
	req, _ := http.NewRequest(http.MethodGet, "http://example.test/x", nil)
	if _, err := rt.RoundTrip(req); err == nil { //nolint:bodyclose // transport error path returns a nil response
		t.Fatal("expected transport error")
	}
	if called {
		t.Fatal("mutate fn ran on the error path")
	}
}
