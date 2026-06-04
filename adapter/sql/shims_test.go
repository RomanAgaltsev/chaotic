package sql_test

import (
	"context"
	"database/sql"
	dbdrv "database/sql/driver"
	"errors"
)

// --- failing driver shim: QueryContext always errors ---
// Used by TestSQLReportsOutcomeToFailureBudget so the wrapped call's outcome
// is a non-nil error, letting the failure budget fill with errors and trip.
// It is also the underlying driver the chaos_off no-op test wraps, so it lives
// in an untagged file that compiles under both build configurations.

type failingDriver struct{}

func (failingDriver) Open(name string) (dbdrv.Conn, error) {
	return &failingConn{}, nil
}

type failingConn struct{}

// Prepare/Close/Begin satisfy driver.Conn
func (c *failingConn) Prepare(query string) (dbdrv.Stmt, error) {
	return nil, errors.New("db down")
}

func (c *failingConn) Begin() (dbdrv.Tx, error) {
	return nil, errors.New("db down")
}

func (c *failingConn) Close() error {
	return nil
}

// QueryContext satisfies driver.QueryerContext and always errors.
func (c *failingConn) QueryContext(_ context.Context, _ string, _ []dbdrv.NamedValue) (dbdrv.Rows, error) {
	return nil, errors.New("db down")
}

func init() {
	sql.Register("failing-shim", &failingDriver{})
}
