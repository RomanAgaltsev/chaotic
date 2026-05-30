// Package grpc provides client and server interceptors that subject gRPC
// calls to chaos. This module is separate from chaotic's main module
// so users who only need HTTP or SQL chaos don't pull in the grpc dep tree.
package grpc

import (
	"context"
	"errors"

	"github.com/ag4r/chaotic/fault"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ag4r/chaotic/engine"
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor that consults
// eng before delegating to the wrapped invoker. If eng is nil or has no rules,
// the interceptor is a near-zero-cost passthrough.
func UnaryClientInterceptor(eng *engine.Engine) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !eng.Enabled() {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		op := engine.Op{
			Kind:   engine.OpGRPCClient,
			Name:   method,
			Method: "unary",
		}
		action := eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			return toStatus(err)
		}
		err := invoker(ctx, method, req, reply, cc, opts...)
		if o, ok := action.(engine.OutcomeReporter); ok {
			o.Outcome(ctx, err)
		}
		return err
	}
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that consults
// eng before delegating to the handler.
func UnaryServerInterceptor(eng *engine.Engine) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !eng.Enabled() {
			return handler(ctx, req)
		}
		op := engine.Op{
			Kind:   engine.OpGRPCServer,
			Name:   info.FullMethod,
			Method: "unary",
		}
		action := eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			return nil, toStatus(err)
		}
		resp, herr := handler(ctx, req)
		if o, ok := action.(engine.OutcomeReporter); ok {
			o.Outcome(ctx, herr)
		}
		return resp, herr
	}
}

// toStatus maps a fault error to a gRPC status.
//   - ErrConnDrop -> codes.Unavailable
//   - status.Status (as error) passes through
//   - everything else -> codes.Internal wrapping the error message.
func toStatus(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return status.Error(codes.Unavailable, "chaotic: conn drop")
	}
	if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown {
		return err
	}
	return status.Error(codes.Internal, err.Error())
}

// StreamClientInterceptor returns a grpc.StreamClientInterceptor that
// consults eng at stream open.
func StreamClientInterceptor(eng *engine.Engine) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if !eng.Enabled() {
			return streamer(ctx, desc, cc, method, opts...)
		}
		op := engine.Op{
			Kind:   engine.OpGRPCClient,
			Name:   method,
			Method: "stream",
		}
		action := eng.Eval(ctx, op)
		if err := action.Before(ctx); err != nil {
			return nil, toStatus(err)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor that
// consults eng before delegating to the handler.
func StreamServerInterceptor(eng *engine.Engine) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !eng.Enabled() {
			return handler(srv, ss)
		}
		op := engine.Op{
			Kind:   engine.OpGRPCServer,
			Name:   info.FullMethod,
			Method: "stream",
		}
		action := eng.Eval(ss.Context(), op)
		if err := action.Before(ss.Context()); err != nil {
			return toStatus(err)
		}
		return handler(srv, ss)
	}
}
