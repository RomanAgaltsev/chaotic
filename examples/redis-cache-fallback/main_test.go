package main

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/RomanAgaltsev/chaotic/adapter/redis"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func TestCacheFallsBackWhenRedisDrops(t *testing.T) {
	mr := miniredis.RunT(t)

	eng := engine.New()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	rc.AddHook(chaosredis.NewHook(eng))

	db := map[string]string{"user:1": "alice"}
	store := NewStore(rc, db)
	ctx := context.Background()

	// Drop every GET: Redis is "down".
	eng.AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.MatchName("get"),
		engine.Always(),
		engine.WithFault(fault.ConnDrop()),
	).Named("redis-down"))

	got, ok := store.Get(ctx, "user:1")
	if !ok || got != "alice" {
		t.Fatalf("Get = %q, %v; want \"alice\", true (served from DB fallback)", got, ok)
	}
	if eng.Hits("redis-down") == 0 {
		t.Fatal("expected the Redis outage rule to have fired")
	}
}
