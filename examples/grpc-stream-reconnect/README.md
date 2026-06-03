# grpc-stream-reconnect

A streaming gRPC client reconnects after a transient `Unavailable`.

`newEngine` installs a `Times(1)` `ConnDrop` rule on `OpGRPCClient`; the gRPC
adapter maps `ConnDrop` to `codes.Unavailable`. `openWithRetry` retries while the
code is `Unavailable`, so the second open succeeds.

This example has its own `go.mod` because it imports the `adapter/grpc`
submodule.