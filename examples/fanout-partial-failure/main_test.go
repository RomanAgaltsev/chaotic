package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func newServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
}

func TestFanOutDropsFaultedBranch(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	got := FanOut(NewClient(), srv.URL, []string{"/a", "/b", "/c"})
	want := []string{"/a", "/c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FanOut = %v, want %v (branch /b is faulted)", got, want)
	}
}

func TestFanOutAllSucceedWithoutChaos(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	got := FanOut(http.DefaultClient, srv.URL, []string{"/a", "/b", "/c"})
	if len(got) != 3 {
		t.Fatalf("FanOut = %v, want all 3 to succeed", got)
	}
}
