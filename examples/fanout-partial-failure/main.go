// Command fanout-partial-failure shows graceful degradation: a fan-out queries
// three backends concurrently, chaos faults exactly one branch (path /b), and
// the aggregator returns the partial result instead of failing the whole
// request.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"

	chaoshttp "github.com/RomanAgaltsev/chaotic/adapter/http"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// FanOut GETs each path through client concurrently and returns the sorted list
// of paths that succeeded. A failed branch is dropped, not fatal.
func FanOut(client *http.Client, base string, paths []string) []string {
	var (
		mu sync.Mutex
		ok []string
		wg sync.WaitGroup
	)
	for _, p := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			resp, err := client.Get(base + p)
			if err != nil {
				return
			}
			_ = resp.Body.Close()
			mu.Lock()
			ok = append(ok, p)
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	slices.Sort(ok)
	return ok
}

// NewClient builds an *http.Client whose transport faults only the /b branch, so
// FanOut must degrade to a partial result.
func NewClient() *http.Client {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.MatchName("/b"),
		engine.WithFault(fault.Error(errors.New("backend b down"))),
	).Named("b-down"))
	return &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	got := FanOut(NewClient(), srv.URL, []string{"/a", "/b", "/c"})
	fmt.Printf("succeeded: %v\n", got)
}
