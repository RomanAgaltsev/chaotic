// Command net-conn-drop demonstrates a read loop that retries through a transient
// connection drop, proven with the chaotic adapter/net wrapper — no real network,
// no Docker (an in-memory net.Pipe stands in for the connection).
package main

import (
	"fmt"
	"net"
	"os"
)

// ReadWithRetry reads up to len(buf) bytes, retrying on error up to attempts
// times, so a transient drop does not surface as a hard failure.
func ReadWithRetry(c net.Conn, buf []byte, attempts int) (int, error) {
	var n int
	var err error
	for range attempts {
		n, err = c.Read(buf)
		if err == nil {
			return n, nil
		}
	}
	return n, err
}

func main() {
	fmt.Fprintln(os.Stdout, "run `go test` in this directory to see a read loop survive a chaos drop")
}
