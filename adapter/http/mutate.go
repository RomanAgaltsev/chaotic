package http

import (
	"net/http"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// MutateResponse returns a fault that runs fn on the *http.Response after a
// successful round-trip, replacing the response the caller observes with fn's
// return value. fn runs only on the success path; on a transport error the
// native error passes through untouched. fn must not eagerly drain resp.Body;
// wrap it lazily if a body transform is needed.
func MutateResponse(fn func(*http.Response) *http.Response) fault.Fault {
	return fault.ResponseMutate(func(v any) any {
		resp, ok := v.(*http.Response)
		if !ok {
			return v
		}
		return fn(resp) //nolint:bodyclose // fn returns the response onward for the caller to consume and close; closing it here would break the round-trip contract
	})
}
