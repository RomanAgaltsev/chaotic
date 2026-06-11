# Raw-conn read retry under chaos

A read loop that retries through a transient connection drop, proven with the
chaotic `adapter/net` wrapper — no real network, no Docker (an in-memory
`net.Pipe` stands in for the connection).

## What it shows

- Wrap any `net.Conn` for chaos: `c := chaosnet.WrapConn(conn, eng)`.
- A rule that `ConnDrop`s the first two reads (`Times(2)`) simulates a transient
  link drop; `ConnDrop` returns a `*net.OpError` and never touches the underlying
  conn, so the retry's next read succeeds.
- The rule's hit counter proves the drop was exercised exactly twice.

## Run it

go test ./…
