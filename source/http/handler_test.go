package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ag4r/chaotic/engine"
	srchttp "github.com/ag4r/chaotic/source/http"
)

const sampleDoc = `rules:
  - name: a
    kinds: [http_client]
    faults: [{type: error, message: x}]
`

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
