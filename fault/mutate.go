package fault

import "context"

// ResponseMutate runs fn on the wrapped call's result after a SUCCESSFUL call,
// returning fn's (possibly modified) value to the adapter. It is a no-op in the
// Before fault chain and never short-circuits other faults in the same rule.
//
// fn receives the adapter's native result as any (*http.Response for the http
// client, the reply message for a gRPC unary call). fn must return a value of
// the same concrete type; a mismatched or nil return is ignored by the adapter,
// which keeps the original result. Prefer the typed helpers where they exist
// (adapter/http.MutateResponse, adapter/grpc.MutateReply).
func ResponseMutate(fn func(any) any) Fault {
	return responseMutateFault{fn: fn}
}

type responseMutateFault struct {
	fn func(any) any
}

// Apply is a no-op: a result mutator does nothing before the wrapped call.
func (responseMutateFault) Apply(context.Context) error { return nil }

// MutateResult runs the user function on the wrapped call's result. A nil fn
// passes the result through unchanged.
func (f responseMutateFault) MutateResult(result any) any {
	if f.fn == nil {
		return result
	}
	return f.fn(result)
}

func (responseMutateFault) Kind() Kind { return KindResponseMutate }
