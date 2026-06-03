package main

import (
	"net/http"
	"testing"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
)

func TestRetryRecovers(t *testing.T) {
	if err := run(); err != nil {
		t.Fatalf("retry did not recover from a single injected fault: %v", err)
	}
}

func TestSingleAttemptSurfacesFault(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	client := &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, newEngine())}
	resp, err := getWithRetry(client, srv.URL, 1)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected the first-attempt fault to surface with attempts=1")
	}
}
