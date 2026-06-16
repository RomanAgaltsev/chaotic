// Command response-mutate demonstrates fault.ResponseMutate via
// adapter/http.MutateResponse: chaos corrupts the body of an otherwise
// successful 200 response, and the client's decode path degrades gracefully
// (returns "unknown") instead of crashing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	chaoshttp "github.com/RomanAgaltsev/chaotic/adapter/http"
	"github.com/RomanAgaltsev/chaotic/engine"
)

type payload struct {
	Name string `json:"name"`
}

// FetchName GETs url through client and returns payload.Name, or "unknown" if
// the body cannot be decoded (the resilience this example exercises).
func FetchName(client *http.Client, url string) string {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "unknown"
	}
	resp, err := client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return "unknown"
	}
	return p.Name
}

// NewClient builds an *http.Client whose transport corrupts successful response
// bodies via MutateResponse, to exercise FetchName's decode-failure fallback.
func NewClient() *http.Client {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpHTTPClient),
		engine.WithFault(chaoshttp.MutateResponse(func(r *http.Response) *http.Response {
			r.Body = io.NopCloser(strings.NewReader("}{ not json"))
			return r
		})),
	))
	return &http.Client{Transport: chaoshttp.WrapTransport(http.DefaultTransport, eng)}
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload{Name: "alice"})
	}))
	defer srv.Close()
	fmt.Fprintln(os.Stdout, FetchName(NewClient(), srv.URL))
}
