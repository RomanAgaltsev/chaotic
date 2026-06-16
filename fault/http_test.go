package fault

import (
	"context"
	"errors"
	"testing"
)

func TestHTTPStatusReturnsSentinel(t *testing.T) {
	f := HTTPStatus(503, "overloaded")
	err := f.Apply(context.Background())
	var hse *HTTPStatusError
	if !errors.As(err, &hse) {
		t.Fatalf("Apply err = %T %v, want *HTTPStatusError", err, err)
	}
	if hse.StatusCode() != 503 {
		t.Fatalf("StatusCode = %d, want 503", hse.StatusCode())
	}
	if hse.Body != "overloaded" {
		t.Fatalf("Body = %q, want %q", hse.Body, "overloaded")
	}
}

func TestHTTPStatusBodyOptional(t *testing.T) {
	var hse *HTTPStatusError
	if !errors.As(HTTPStatus(429).Apply(context.Background()), &hse) {
		t.Fatal("expected *HTTPStatusError")
	}
	if hse.Body != "" {
		t.Fatalf("Body = %q, want empty (adapter substitutes default)", hse.Body)
	}
}

func TestHTTPStatusKind(t *testing.T) {
	if got := KindOf(HTTPStatus(500)); got != KindHTTPStatus {
		t.Fatalf("KindOf = %v, want KindHTTPStatus", got)
	}
}

func TestHeaderInjectReturnsSentinel(t *testing.T) {
	var hf *HeaderFaultError
	if !errors.As(HeaderInject("X-Trace", "abc").Apply(context.Background()), &hf) {
		t.Fatal("expected *HeaderFaultError")
	}
	if hf.Strip || hf.Key != "X-Trace" || hf.Value != "abc" {
		t.Fatalf("hf = %+v, want set X-Trace:abc", hf)
	}
}

func TestHeaderStripReturnsSentinel(t *testing.T) {
	var hf *HeaderFaultError
	if !errors.As(HeaderStrip("X-Trace").Apply(context.Background()), &hf) {
		t.Fatal("expected *HeaderFaultError")
	}
	if !hf.Strip || hf.Key != "X-Trace" {
		t.Fatalf("hf = %+v, want strip X-Trace", hf)
	}
}

func TestHeaderFaultKind(t *testing.T) {
	if got := KindOf(HeaderStrip("X")); got != KindHeader {
		t.Fatalf("KindOf = %v, want KindHeader", got)
	}
}
