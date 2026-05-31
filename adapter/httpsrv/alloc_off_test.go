//go:build chaos_off

package httpsrv_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ag4r/chaotic/adapter/httpsrv"
	"github.com/ag4r/chaotic/engine"
)

type noopRW struct {
	h http.Header
}

func (w noopRW) Header() http.Header {
	return w.h
}

func (w noopRW) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w noopRW) WriteHeader(int) {}

func TestZeroAllocUnderChaosOff(t *testing.T) {
	h := httpsrv.Middleware(engine.New())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rw := noopRW{h: http.Header{}}
	avg := testing.AllocsPerRun(100, func() { h.ServeHTTP(rw, req) })
	if avg != 0 {
		t.Fatalf("allocs/op = %v, want 0 under chaos_off", avg)
	}
}
