package http_test

import "net/http"

// roundTripperFunc adapts a function to http.RoundTripper. It is shared by the
// behavior tests (!chaos_off) and the zero-alloc test (chaos_off), so it lives
// in an untagged file that compiles under both build configurations.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
