# response-mutate

Demonstrates [`fault.ResponseMutate`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/fault#ResponseMutate)
via [`adapter/http.MutateResponse`](https://pkg.go.dev/github.com/RomanAgaltsev/chaotic/adapter/http#MutateResponse).

A test server returns a valid `200 OK` JSON body (`{"name":"alice"}`). The chaos
rule rewrites the **successful** response body to malformed JSON before the
client sees it, exercising the caller's decode-failure fallback: `FetchName`
returns `"unknown"` instead of panicking or returning a bogus value.

```bash
go run .        # prints: unknown
go test ./...   # asserts the degraded path and the clean (no-chaos) path
