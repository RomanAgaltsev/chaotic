package grpc_test

import (
	"context"
	"fmt"

	chaosgrpc "github.com/ag4r/chaotic/adapter/grpc"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
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
