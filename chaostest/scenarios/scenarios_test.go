package scenarios_test

import (
	"context"
	"testing"

	"github.com/ag4r/chaotic/chaostest"
	"github.com/ag4r/chaotic/chaostest/scenarios"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func fire(t *testing.T, eng *engine.Engine, op engine.Op) error {
	t.Helper()
	act := eng.Eval(context.Background(), op)
	err := act.Before(context.Background())
	_ = act.After(context.Background())
	return err
}

func TestDatabaseOutageCascade(t *testing.T) {
	eng := chaostest.New(t)
	scenarios.DatabaseOutageCascade(eng, scenarios.WithCount(3))

	// The first 3 SQL calls drop.
	drops := 0
	for range 3 {
		if err := fire(t, eng, engine.Op{Kind: engine.OpSQL, Name: "SELECT"}); err != nil {
			drops++
		}
	}
	if drops != 3 {
		t.Fatalf("got %d drops in the outage window, want 3", drops)
	}
	chaostest.AssertHits(t, eng, "scenarios/db-outage", 3)
}

func TestThunderingHerdAfterDeploy(t *testing.T) {
	eng := chaostest.New(t)
	// errorRate 1.0 makes the assertion deterministic.
	scenarios.ThunderingHerdAfterDeploy(eng, scenarios.WithErrorRate(1))

	got503 := 0
	for range 4 {
		if err := fire(t, eng, engine.Op{Kind: engine.OpHTTPServer, Name: "/checkout"}); err != nil {
			got503++
		}
	}
	if got503 != 4 {
		t.Fatalf("got %d failed requests at errorRate=1, want 4", got503)
	}
}

func TestSlowLeaderElection(t *testing.T) {
	eng := chaostest.New(t)
	scenarios.SlowLeaderElection(eng, scenarios.WithCount(2))

	for range 2 {
		// latency Before returns nil (it just sleeps), so we assert via hit count.
		_ = fire(t, eng, engine.Op{Kind: engine.OpRedis, Name: "SET"})
	}
	chaostest.AssertHits(t, eng, "scenarios/slow-election", 2)
}

func TestPartialNetworkPartition(t *testing.T) {
	eng := chaostest.New(t)
	scenarios.PartialNetworkPartition(eng, scenarios.WithErrorRate(1)) // 1.0 = deterministic

	dropped := 0
	for range 3 {
		if err := fire(t, eng, engine.Op{Kind: engine.OpGRPCClient, Name: "/svc/Method"}); err != nil {
			dropped++
		}
	}
	if dropped != 3 {
		t.Fatalf("got %d dropped calls at errorRate=1, want 3", dropped)
	}
	_ = fault.ErrConnDrop
}

func TestAWSRegionFailover(t *testing.T) {
	eng := chaostest.New(t)
	scenarios.AWSRegionFailover(eng, scenarios.WithCount(3))

	// First 3 AWS calls drop (region unreachable).
	drops := 0
	for range 3 {
		if err := fire(t, eng, engine.Op{Kind: engine.OpAWS, Name: "dynamodb.GetItem"}); err != nil {
			drops++
		}
	}
	if drops != 3 {
		t.Fatalf("got %d drops in the outage window, want 3", drops)
	}
	// Next 3 calls are latency-only (no error returned by Before).
	for range 3 {
		if err := fire(t, eng, engine.Op{Kind: engine.OpAWS, Name: "dynamodb.GetItem"}); err != nil {
			t.Fatalf("failover-lag window should not return an error, got %v", err)
		}
	}
	chaostest.AssertHits(t, eng, "scenarios/aws-region-down", 3)
	chaostest.AssertHits(t, eng, "scenarios/aws-failover-lag", 3)
}
