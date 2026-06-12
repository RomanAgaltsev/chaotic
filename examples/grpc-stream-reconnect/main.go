// Command grpc-stream-reconnect shows a streaming RPC client retrying stream
// creation after chaos injects a transient Unavailable on the first open.
// Run with `go run .`.
package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chaosgrpc "github.com/RomanAgaltsev/chaotic/adapter/grpc"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// newEngine fails only the first stream open, then becomes inert.
func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.Times(1),
		engine.WithFault(fault.ConnDrop()),
	).Named("stream-flap"))
}

type nopClientStream struct {
	grpc.ClientStream
}

// openStream runs the chaos interceptor in front of a streamer that would
// normally succeed.
func openStream(intc grpc.StreamClientInterceptor) (grpc.ClientStream, error) {
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		return nopClientStream{}, nil
	}
	desc := &grpc.StreamDesc{StreamName: "Feed", ServerStreams: true}
	return intc(context.Background(), desc, nil, "/demo.Service/Feed", streamer)
}

// openWithRetry retries opening the stream while the error is a transient
// Unavailable, up to attempts times.
func openWithRetry(intc grpc.StreamClientInterceptor, attempts int) (grpc.ClientStream, error) {
	var err error
	for range attempts {
		var s grpc.ClientStream
		if s, err = openStream(intc); err == nil {
			return s, nil
		}
		if status.Code(err) != codes.Unavailable {
			return nil, err
		}
	}
	return nil, err
}

func main() {
	intc := chaosgrpc.StreamClientInterceptor(newEngine())
	if _, err := openWithRetry(intc, 3); err != nil {
		fmt.Println("FAILED:", err)
		return
	}
	fmt.Println("stream established after reconnect despite injected Unavailable")
}
