//go:build chaos_off

package redis_test

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/ag4r/chaotic/adapter/redis"
	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	next := func(context.Context, goredis.Cmder) error { return nil }
	ph := chaosredis.NewHook(engine.New()).ProcessHook(next)
	ctx := context.Background()
	cmd := goredis.NewCmd(ctx, "get", "k")
	avg := testing.AllocsPerRun(100, func() {
		_ = ph(ctx, cmd)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
