//go:build !chaos_off

package redis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/RomanAgaltsev/chaotic/adapter/redis"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestHookOnRealClient(t *testing.T) {
	mr := miniredis.RunT(t)
	if err := mr.Set("greeting", "hello"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	eng := engine.New()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	rc.AddHook(chaosredis.NewHook(eng))

	ctx := context.Background()

	// With no rules, the command succeeds normally.
	if got, err := rc.Get(ctx, "greeting").Result(); err != nil || got != "hello" {
		t.Fatalf("Get = %q, %v; want \"hello\", nil", got, err)
	}

	// Add a rule that fails GET, then assert the error surfaces through the client.
	sentinel := errors.New("redis unavailable")
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.MatchName("get"),
		engine.Always(),
		engine.WithFault(fault.Error(sentinel)),
	).Named("get-fail"))

	if _, err := rc.Get(ctx, "greeting").Result(); !errors.Is(err, sentinel) {
		t.Fatalf("Get err = %v, want sentinel through the client", err)
	}
}
