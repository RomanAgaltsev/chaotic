//go:build pgxintegration

package main

import (
	"context"
	"os"
	"testing"
)

func TestExecRetryRecovers(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run the pgx-pool integration example test")
	}
	if err := run(context.Background(), dsn); err != nil {
		t.Fatalf("exec retry did not recover from the injected fault: %v", err)
	}
}
