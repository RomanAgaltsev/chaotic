//go:build !chaos_off

package grpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
	"github.com/RomanAgaltsev/chaotic/engine"
)

type mreply struct{ Body string }

func TestUnaryClientMutateReplyRewritesReply(t *testing.T) {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(chaosgrpc.MutateReply(func(r *mreply) *mreply {
			r.Body = "MUTATED"
			return r
		})),
	))
	intercept := chaosgrpc.UnaryClientInterceptor(eng)
	got := &mreply{}
	invoker := func(ctx context.Context, method string, req, rep any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		rep.(*mreply).Body = "ok"
		return nil
	}
	if err := intercept(context.Background(), "/svc/M", nil, got, nil, invoker); err != nil {
		t.Fatal(err)
	}
	if got.Body != "MUTATED" {
		t.Fatalf("reply.Body = %q, want MUTATED", got.Body)
	}
}

func TestUnaryClientMutateReplyNotRunOnError(t *testing.T) {
	mutated := false
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(chaosgrpc.MutateReply(func(r *mreply) *mreply {
			mutated = true
			return r
		})),
	))
	intercept := chaosgrpc.UnaryClientInterceptor(eng)
	invoker := func(ctx context.Context, method string, req, rep any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return errors.New("backend down")
	}
	_ = intercept(context.Background(), "/svc/M", nil, &mreply{}, nil, invoker)
	if mutated {
		t.Fatal("mutate fn ran on the error path")
	}
}
