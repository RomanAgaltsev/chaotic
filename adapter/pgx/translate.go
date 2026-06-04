package pgx

import (
	"errors"
	"io"
	"net"

	"github.com/ag4r/chaotic/fault"
)

// translate maps a fault-pipeline error into pgx's native error model.
//
//   - fault.ErrConnDrop → *net.OpError wrapping io.ErrUnexpectedEOF. pgconn's
//     wire-read path treats any net.Error during a message read as "connection
//     broken", which causes pgxpool to evict the conn on next Release. This
//     matches what a real lost connection looks like to user code.
//   - Any other error is returned verbatim. Users who want a
//     specific pgx-shaped error (e.g. *pgconn.PgError) construct it themselves
//     and pass it to fault.Error(...).
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fault.ErrConnDrop) {
		return &net.OpError{
			Op:  "read",
			Net: "tcp",
			Err: io.ErrUnexpectedEOF,
		}
	}
	return err
}
