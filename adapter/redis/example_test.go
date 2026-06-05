//go:build !chaos_off

package redis_test

import (
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	chaosredis "github.com/ag4r/chaotic/adapter/redis"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleNewHook() {
	// Fail the first GET, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpRedis),
		engine.MatchName("get"),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("cache down"))),
	).Named("redis-flap"))

	// next stands in for the real go-redis command execution.
	next := func(context.Context, goredis.Cmder) error { return nil }
	process := chaosredis.NewHook(eng).ProcessHook(next)

	get := func() error {
		cmd := goredis.NewCmd(context.Background(), "get", "k")
		return process(context.Background(), cmd)
	}

	fmt.Println("attempt 1:", get())
	fmt.Println("attempt 2:", get())
	// Output:
	// attempt 1: cache down
	// attempt 2: <nil>
}
