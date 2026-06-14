package grpc

import "github.com/RomanAgaltsev/chaotic/fault"

// MutateReply returns a fault that runs fn on the unary reply message of type T
// after a successful call, mutating the reply the caller observes. T is the
// caller's response message type (e.g. *pb.GetReply). fn runs only on the
// success path. The reply is typically mutated in place; fn should return it.
// A reply whose dynamic type is not T is left unchanged.
func MutateReply[T any](fn func(T) T) fault.Fault {
	return fault.ResponseMutate(func(v any) any {
		t, ok := v.(T)
		if !ok {
			return v
		}
		return fn(t)
	})
}
