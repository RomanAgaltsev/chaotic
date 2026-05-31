//go:build chaos_off

package http_test

import (
	"net/http"
	"testing"

	chaoshttp "github.com/ag4r/chaotic/adapter/http"
	"github.com/ag4r/chaotic/engine"
)

func TestZeroAllocUnderChaosOff(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Body: http.NoBody}
	noop := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return resp, nil
	})
	rt := chaoshttp.WrapTransport(noop, engine.New())
	req, _ := http.NewRequest("GET", "http://x/y", nil)
	avg := testing.AllocsPerRun(100, func() {
		_, _ = rt.RoundTrip(req)
	})
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
