//go:build !chaos_off

package main

import (
	"context"
	"testing"

	"github.com/RomanAgaltsev/chaotic/chaos"
)

func TestRetrySurvivesInjectedFault(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), newEngine())
	if err := publishWithRetry(ctx, "order-42", 3); err != nil {
		t.Fatalf("retry did not recover from a single injected fault: %v", err)
	}
}

func TestSingleAttemptFails(t *testing.T) {
	ctx := chaos.WithEngine(context.Background(), newEngine())
	if err := publishWithRetry(ctx, "order-42", 1); err == nil {
		t.Fatal("expected the first-attempt fault to surface with attempts=1")
	}
}
