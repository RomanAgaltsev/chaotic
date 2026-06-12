# slow-body-read

Demonstrates `adapter/io`. `ReadBody` reads a response body through a
`chaosio.WrapReader`. A `fault.Truncate(4)` rule cuts the body mid-JSON; the test
asserts the consumer surfaces a clean parse error instead of panicking
(cookbook recipe #17). Swap in `fault.SlowReader(rate)` with a read deadline to
exercise the "timeout while reading the body" path (recipe #15).

Run: `go test ./...`
