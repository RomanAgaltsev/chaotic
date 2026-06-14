package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload{Name: "alice"})
	}))
}

func TestFetchNameDegradesOnCorruptBody(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	if got := FetchName(NewClient(), srv.URL); got != "unknown" {
		t.Fatalf("FetchName = %q, want unknown (corrupted body should fail to decode)", got)
	}
}

func TestFetchNameWithoutChaos(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	if got := FetchName(http.DefaultClient, srv.URL); got != "alice" {
		t.Fatalf("FetchName = %q, want alice", got)
	}
}
