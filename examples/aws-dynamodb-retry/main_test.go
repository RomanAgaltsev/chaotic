package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	chaosaws "github.com/ag4r/chaotic/adapter/aws"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestGetItemSurvivesOutageViaSDKRetry(t *testing.T) {
	// A local "DynamoDB": always returns a valid empty GetItem JSON response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	eng := engine.New()
	cfg := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("AKID", "SECRET", ""),
		RetryMaxAttempts: 3,
	}
	// Install chaos at Finalize so the SDK's retryer retries injected failures.
	chaosaws.AppendChaosMiddleware(&cfg, eng)

	ddb := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
	})

	// Drop the first two attempts: a transient outage. The SDK's default retryer
	// (max 3 attempts) retries them; the 3rd attempt reaches the httptest server.
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpAWS),
		engine.Times(2),
		engine.WithFault(fault.ConnDrop()),
	).Named("outage"))

	out, err := GetItem(context.Background(), ddb, "widgets", "1")
	if err != nil {
		t.Fatalf("GetItem failed despite SDK retries: %v", err)
	}
	if out == nil {
		t.Fatal("GetItem returned nil output")
	}
	if got := eng.Hits("outage"); got != 2 {
		t.Fatalf("outage fired %d times, want 2 (the SDK retried the two ConnDrops)", got)
	}
}
