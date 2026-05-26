package grpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chaosgrpc "github.com/ag4r/chaotic/adapter/grpc"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// nopInvoker is a grpc.UnaryInvoker that records invocation and returns nil.
type nopInvoker struct {
	called bool
}

func (n *nopInvoker) invoke(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	n.called = true
	return nil
}

func TestUnaryClientInterceptorErrorShortCircuits(t *testing.T) {
	target := errors.New("boom")
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(fault.Error(target)),
	))
	intercept := chaosgrpc.UnaryClientInterceptor(e)
	inv := &nopInvoker{}
	err := intercept(context.Background(), "/svc/Method", nil, nil, nil, inv.invoke)
	if inv.called {
		t.Fatal("invoker was called despite fault")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("status = %s, want Internal", st.Code())
	}
}

func TestUnaryClientInterceptorLatencyDelaysInvoker(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(fault.Latency(400*time.Millisecond)),
	))
	intercept := chaosgrpc.UnaryClientInterceptor(e)
	inv := &nopInvoker{}
	start := time.Now()
	if err := intercept(context.Background(), "/svc/Method", nil, nil, nil, inv.invoke); err != nil {
		t.Fatal(err)
	}
	if !inv.called {
		t.Fatal("invoker was not called after latency")
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatal("returned too quickly")
	}
}

func TestUnaryClientInterceptorConnDropMapsToUnavailable(t *testing.T) {
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(fault.ConnDrop()),
	))
	intercept := chaosgrpc.UnaryClientInterceptor(e)
	inv := &nopInvoker{}
	err := intercept(context.Background(), "/svc/Method", nil, nil, nil, inv.invoke)
	st, _ := status.FromError(err)
	if st.Code() != codes.Unavailable {
		t.Fatalf("status = %s, want Unavailable", st.Code())
	}
}

func TestUnaryClientInterceptorPassthroughWhenEmpty(t *testing.T) {
	e := engine.New() // no rules
	intercept := chaosgrpc.UnaryClientInterceptor(e)
	inv := &nopInvoker{}
	if err := intercept(context.Background(), "/svc/Method", nil, nil, nil, inv.invoke); err != nil {
		t.Fatal(err)
	}
	if !inv.called {
		t.Fatal("invoker should have been called on passthrough")
	}
}

func TestUnaryServerInterceptorErrorShortCircuits(t *testing.T) {
	target := errors.New("boom")
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCServer),
		engine.WithFault(fault.Error(target)),
	))
	intercept := chaosgrpc.UnaryServerInterceptor(e)
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	}
	_, err := intercept(context.Background(), nil, info, handler)
	if handlerCalled {
		t.Fatal("handler called despite fault")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("status = %s, want Internal", st.Code())
	}
}

func TestUnaryServerInterceptorPassthroughWhenEmpty(t *testing.T) {
	e := engine.New()
	intercept := chaosgrpc.UnaryServerInterceptor(e)
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}
	resp, err := intercept(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatal(err)
	}
	if !called || resp != "ok" {
		t.Fatalf("handler not invoked correctly: called=%v resp=%v", called, resp)
	}
}

// fakeServerStream is a minimal grpc.ServerStream stub for direct testing
// of the stream server interceptor.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}

func TestStreamClientInterceptorErrorShortCircuitsAtOpen(t *testing.T) {
	target := errors.New("boom")
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCClient),
		engine.WithFault(fault.Error(target)),
	))
	intercept := chaosgrpc.StreamClientInterceptor(e)
	desc := &grpc.StreamDesc{
		StreamName:    "Method",
		ClientStreams: true,
		ServerStreams: true,
	}
	streamerCalled := false
	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		streamerCalled = true
		return nil, nil
	}
	_, err := intercept(context.Background(), desc, nil, "/svc/Method", streamer)
	if streamerCalled {
		t.Fatal("streamer was called despite fault")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("status = %s, want Internal", st.Code())
	}
}

func TestStreamClientInterceptorPassthroughWhenEmpty(t *testing.T) {
	e := engine.New()
	intercept := chaosgrpc.StreamClientInterceptor(e)
	desc := &grpc.StreamDesc{StreamName: "Method"}
	called := false
	streamer := func(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		return nil, nil
	}
	if _, err := intercept(context.Background(), desc, nil, "/svc/Method", streamer); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("streamer should have been called")
	}
}

func TestStreamServerInterceptorErrorShortCircuits(t *testing.T) {
	target := errors.New("boom")
	e := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpGRPCServer),
		engine.WithFault(fault.Error(target)),
	))
	intercept := chaosgrpc.StreamServerInterceptor(e)
	info := &grpc.StreamServerInfo{FullMethod: "/svc/Method"}
	ss := &fakeServerStream{ctx: context.Background()}
	handlerCalled := false
	handler := func(srv any, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}
	err := intercept(nil, ss, info, handler)
	if handlerCalled {
		t.Fatal("handler called despite fault")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("status = %s, want Internal", st.Code())
	}
}
