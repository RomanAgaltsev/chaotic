package fault

import (
	"context"
	"fmt"
)

// HTTPStatusError is the sentinel an HTTPStatus fault returns. The HTTP adapters
// detect it via errors.As and render the status instead of the generic 500.
// The fault package stays free of net/http: this carries only the code and an
// optional body string.
type HTTPStatusError struct {
	Code int
	Body string // empty => the adapter substitutes http.StatusText(Code)
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("chaotic: http status %d", e.Code)
}

// StatusCode reports the HTTP status code the fault should render.
func (e *HTTPStatusError) StatusCode() int {
	return e.Code
}

// HTTPStatus returns a fault that makes the HTTP adapters render the given
// status code: adapter/httpsrv writes it, adapter/http synthesizes a response
// carrying it. Body is optional. When omitted the adapter uses http.StatusText.
func HTTPStatus(code int, body ...string) Fault {
	f := httpStatusFault{code: code}
	if len(body) > 0 {
		f.body = body[0]
	}
	return f
}

type httpStatusFault struct {
	code int
	body string
}

func (h httpStatusFault) Apply(_ context.Context) error {
	return &HTTPStatusError{
		Code: h.code,
		Body: h.body,
	}
}

func (h httpStatusFault) Kind() Kind {
	return KindHTTPStatus
}

// HeaderFaultError is the sentinel a header fault returns. Adapters detect it via
// errors.As, apply the mutation to the headers flowing toward the code under
// test (request headers on the server, response headers on the client), then
// let the wrapped call proceed - it does NOT abort the call.
//
// Because the engine fires at most one rule per Op and rule's fault chain
// short-circuits on the first sentinel, only one header fault applies per rule
// (a preceding Latency is fine, it returns nil and continues). To mutate
// several headers, use several rules, or wait for a future batch helper.
type HeaderFaultError struct {
	Strip bool // true => delete Key, false => set Key to Value
	Key   string
	Value string
}

func (*HeaderFaultError) Error() string {
	return "chaotic: header mutation"
}

// HeaderInject returns a fault that sets header key to value on the headers
// flowing toward the code under test.
func HeaderInject(key, value string) Fault {
	return headerFault{
		hf: HeaderFaultError{
			Key:   key,
			Value: value,
		},
	}
}

// HeaderStrip returns a fault that deletes header key from the headers flowing
// toward the code under test.
func HeaderStrip(key string) Fault {
	return headerFault{
		hf: HeaderFaultError{
			Strip: true,
			Key:   key,
		},
	}
}

type headerFault struct {
	hf HeaderFaultError
}

func (h headerFault) Apply(context.Context) error {
	hf := h.hf // copy so each Apply returns a distinct sentinel
	return &hf
}

func (headerFault) Kind() Kind {
	return KindHeader
}
