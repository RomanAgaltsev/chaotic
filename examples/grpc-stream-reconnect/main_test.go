//go:build !chaos_off

package main

import (
	"testing"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
)

func TestStreamReconnects(t *testing.T) {
	intc := chaosgrpc.StreamClientInterceptor(newEngine())
	if _, err := openWithRetry(intc, 3); err != nil {
		t.Fatalf("stream did not reconnect after a single injected Unavailable: %v", err)
	}
}

func TestSingleAttemptFails(t *testing.T) {
	intc := chaosgrpc.StreamClientInterceptor(newEngine())
	if _, err := openWithRetry(intc, 1); err == nil {
		t.Fatal("expected the first-open Unavailable to surface with attempts=1")
	}
}
