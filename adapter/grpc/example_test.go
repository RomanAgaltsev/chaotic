//go:build !chaos_off

package grpc_test

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func ExampleUnaryClientInterceptor() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.Times(1),
		engine.WithFault(fault.ConnDrop()),
	).Named("grpc-flap"))

	intc := chaosgrpc.UnaryClientInterceptor(eng)

	// A fake invoker that would normally succeed.
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		return nil
	}
	call := func() error {
		return intc(context.Background(), "/demo.Service/Get", nil, nil, nil, invoker)
	}

	fmt.Println("attempt 1 code:", status.Code(call())) // ConnDrop -> Unavailable
	fmt.Println("attempt 2 code:", status.Code(call())) // exhausted -> OK
	// Output:
	// attempt 1 code: Unavailable
	// attempt 2 code: OK
}
