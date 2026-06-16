// Command db-conn-pool shows database/sql discarding a poisoned connection and
// transparently retrying on a fresh one when chaos injects a connection drop.
// Run with `go run .`.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	chaossql "github.com/RomanAgaltsev/chaotic/adapter/sql"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

type countingDriver struct{ opens atomic.Int64 }

func (d *countingDriver) Open(string) (driver.Conn, error) {
	d.opens.Add(1)
	return countingConn{}, nil
}

type countingConn struct{}

func (countingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (countingConn) Close() error                        { return nil }
func (countingConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (countingConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func newEngine() *engine.Engine {
	return engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.Times(1), // poison exactly the first connection
		engine.WithFault(fault.ConnDrop()),
	).Named("poison-conn"))
}

// regSeq keeps driver names unique so run can be called multiple times (main +
// tests) without a duplicate sql.Register panic.
var regSeq atomic.Int64

// run opens a chaos-wrapped DB, runs one Exec, and reports how many physical
// connections the underlying driver had to open. The first conn is poisoned by
// ConnDrop (mapped to driver.ErrBadConn), so database/sql retries on a fresh
// conn: two opens, one successful Exec.
func run() (opens int64, err error) {
	id := regSeq.Add(1)
	base := fmt.Sprintf("counting-%d", id)
	drv := &countingDriver{}
	sql.Register(base, drv)
	chaossql.Register("chaos:"+base, base, newEngine())

	db, err := sql.Open("chaos:"+base, "")
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	_, err = db.ExecContext(context.Background(), "UPDATE ledger SET v = v + 1")
	return drv.opens.Load(), err
}

func main() {
	opens, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED:", err)
		return
	}
	fmt.Fprintf(os.Stdout, "exec succeeded; driver opened %d connections (1 poisoned, 1 healthy)\n", opens)
}
