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

func TestPerRulePutAndDelete(t *testing.T) {
	eng := engine.New()
	h := srchttp.New(eng, srchttp.WithWritable(true))

	// PUT a single rule from a terms string; URL name is authoritative.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/rules/flaky", strings.NewReader(`kind(http_client)=error("boom")`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body.String())
	}
	op := engine.Op{Kind: engine.OpHTTPClient, Name: "/x"}
	if eng.Eval(context.Background(), op).Before(context.Background()) == nil {
		t.Fatal("the PUT rule should fire")
	}
	if eng.Hits("flaky") == 0 {
		t.Fatal("rule should be named after the URL, not the body")
	}

	// DELETE it; it stops firing.
	recd := httptest.NewRecorder()
	h.ServeHTTP(recd, httptest.NewRequest(http.MethodDelete, "/rules/flaky", nil))
	if recd.Code != http.StatusNoContent {
		t.Fatalf("DELETE = %d", recd.Code)
	}
	if eng.Enabled() {
		t.Fatal("engine should be empty after deleting the only rule")
	}
}

func TestPerRulePutRejectsMultiAndInvalid(t *testing.T) {
	h := srchttp.New(engine.New(), srchttp.WithWritable(true))

	multi := httptest.NewRecorder()
	h.ServeHTTP(multi, httptest.NewRequest(http.MethodPut, "/rules/x",
		strings.NewReader(`kind(sql)=conndrop; kind(redis)=conndrop`)))
	if multi.Code != http.StatusBadRequest {
		t.Fatalf("multi-rule PUT = %d, want 400", multi.Code)
	}

	bad := httptest.NewRecorder()
	h.ServeHTTP(bad, httptest.NewRequest(http.MethodPut, "/rules/x", strings.NewReader(`nonsense(`)))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid PUT = %d, want 400", bad.Code)
	}
}

func TestPerRuleWritesNeedWritable(t *testing.T) {
	h := srchttp.New(engine.New()) // read-only
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/rules/x", strings.NewReader(`kind(sql)=conndrop`)))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT on read-only = %d, want 405", rec.Code)
	}
}
