package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	srchttp "github.com/RomanAgaltsev/chaotic/source/http"
)

const sampleDoc = `rules:
  - name: a
    kinds: [http_client]
    faults: [{type: error, message: x}]
`

func post(t *testing.T, h http.Handler, body string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST setup = %d %s", rec.Code, rec.Body.String())
	}
}

func TestPostInstallsAndGetReturnsDoc(t *testing.T) {
	eng := engine.New()
	h := srchttp.New(eng, srchttp.WithWritable(true))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(sampleDoc)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST code = %d, want 204", rec.Code)
	}
	if !eng.Enabled() {
		t.Fatal("engine not enabled after POST")
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(rec2.Body.String(), "name: a") {
		t.Fatalf("GET body = %s", rec2.Body.String())
	}
}

func TestReadOnlyRejectsPost(t *testing.T) {
	h := srchttp.New(engine.New())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("rules: []")))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d, want 405", rec.Code)
	}
}

func TestAuthRejectsMissingToken(t *testing.T) {
	h := srchttp.New(engine.New(), srchttp.WithAuth(func(tok string) bool { return tok == "secret" }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestListAndCountRoutes(t *testing.T) {
	eng := engine.New()
	h := srchttp.New(eng, srchttp.WithWritable(true))
	post(t, h, sampleDoc) // installs rule "a"

	// Fire rule "a" once.
	if err := eng.Eval(context.Background(), engine.Op{Kind: engine.OpHTTPClient, Name: "/x"}).Before(context.Background()); err == nil {
		t.Fatal("rule a should have fired")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rules", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"name":"a"`) {
		t.Fatalf("GET /rules = %d %s", rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/rules/a/count", nil))
	if rec2.Code != http.StatusOK || strings.TrimSpace(rec2.Body.String()) != "1" {
		t.Fatalf("GET /rules/a/count = %d %q, want 200 \"1\"", rec2.Code, rec2.Body.String())
	}

	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/rules/nope/count", nil))
	if rec3.Code != http.StatusNotFound {
		t.Fatalf("unknown count = %d, want 404", rec3.Code)
	}
}
