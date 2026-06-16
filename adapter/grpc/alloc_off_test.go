//go:build chaos_off

package grpc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
	"github.com/RomanAgaltsev/chaotic/engine"
)

func TestZeroAllocUnderChaosOffUnaryClient(t *testing.T) {
	interceptor := chaosgrpc.UnaryClientInterceptor(engine.New())
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}
	ctx := context.Background()
	avg := testing.AllocsPerRun(100, func() {
		_ = interceptor(ctx, "/svc/M", nil, nil, nil, invoker)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
