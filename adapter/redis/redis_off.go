//go:build chaos_off

// Package redis (chaos_off build): NewHook returns a passthrough that adds no
// behavior and no allocation to any go-redis path.
package redis

import (
	"github.com/RomanAgaltsev/chaotic/engine"

	goredis "github.com/redis/go-redis/v9"
)

func NewHook(_ *engine.Engine) goredis.Hook {
	return passHook{}
}

type passHook struct{}

func (passHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (passHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return next
}

func (passHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return next
}
