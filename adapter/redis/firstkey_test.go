//go:build !chaos_off

package redis

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

func TestFirstKey(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		cmd  goredis.Cmder
		want string
	}{
		{"get has key", goredis.NewCmd(ctx, "get", "user:1"), "user:1"},
		{"setex has key", goredis.NewCmd(ctx, "setex", "k", 10, "v"), "k"},
		{"ping is keyless", goredis.NewCmd(ctx, "ping"), ""},
		{"non-string second arg", goredis.NewCmd(ctx, "get", 42), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstKey(tt.cmd); got != tt.want {
				t.Fatalf("firstKey = %q, want %q", got, tt.want)
			}
		})
	}
}
