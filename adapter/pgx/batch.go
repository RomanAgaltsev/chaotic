package pgx

import (
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// chaosBatch satisfies pgxv5.BatchResults and yields the carried error from
// every method. Used when a chaos rule fires on SendBatch — per spec §5.4,
// faults at batch-level fire once at batch open and surface uniformly for
// each subsequent iterator call.
type chaosBatch struct {
	err error
}

func (b chaosBatch) Exec() (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, b.err
}

func (b chaosBatch) Query() (pgxv5.Rows, error) {
	return nil, b.err
}

func (b chaosBatch) QueryRow() pgxv5.Row {
	return chaosRow(b)
}

func (b chaosBatch) Close() error {
	return b.err
}
