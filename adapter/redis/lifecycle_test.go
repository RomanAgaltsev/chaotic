//go:build !chaos_off

package redis_test

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/ag4r/chaotic/adapter/redis"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// A WithMaxConcurrent slot must be released after each command, or chaos
// silently stops once the cap is reached. fault.Latency(0) returns nil from
// Before, so the slot is freed only by ProcessHook running After.
func TestProcessHookReleasesMaxConcurrentSlot(t *testing.T) {
	eng := engine.New(engine.WithMaxConcurrent(1)).AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.Always(),
		engine.WithFault(fault.Latency(0)),
	).Named("lat"))

	ph := chaosredis.NewHook(eng).ProcessHook(
		func(context.Context, goredis.Cmder) error { return nil },
	)
	ctx := context.Background()
	for range 3 {
		cmd := goredis.NewCmd(ctx, "get", "k")
		if err := ph(ctx, cmd); err != nil {
			t.Fatalf("ph err = %v", err)
		}
	}
	if got := eng.Hits("lat"); got != 3 {
		t.Fatalf("rule fired %d/3 sequential commands; the max-concurrent slot is leaking", got)
	}
}
