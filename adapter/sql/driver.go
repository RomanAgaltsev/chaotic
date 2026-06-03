//go:build !chaos_off

// Package sql wraps a database/sql driver so calls are subject to chaos.
// Register a chaos driver name that points at an existing registered
// driver, then sql.Open chaos name.
//
// Example:
//
//	chaossql.Register("chaos:postgres", "postgres", eng)
//	db, _ := sql.Open("chaos:postgres", dsn)
package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

// Register registers a driver that wraps wrappedDriverName with chaos.
// The wrapped driver must already be registered with database/sql.
// Subsequent calls to sql.Open(driverName, dsn) get chaos-wrapped driver.
//
// Panics if driverName is already registered (per database/sql semantics).
func Register(driverName, wrappedDriverName string, eng *engine.Engine) {
	target := findDriver(wrappedDriverName)
	if target == nil {
		panic(fmt.Sprintf("chaotic: unknown wrapped driver %q", wrappedDriverName))
	}
	sql.Register(driverName, &chaosDriver{
		wrapped: target,
		eng:     eng,
	})
}

func findDriver(name string) driver.Driver {
	// database/sql doesn't expose drivers directly. Open a throwaway DB to
	// extract the driver. sql.Open is lazy - it does not dial - so an empty
	// dsn is fine as long as the driver is registered.
	db, err := sql.Open(name, "")
	if err != nil {
		return nil
	}
	d := db.Driver()
	_ = db.Close()
	return d
}

type chaosDriver struct {
	wrapped driver.Driver
	eng     *engine.Engine
}

func (d *chaosDriver) Open(name string) (driver.Conn, error) {
	c, err := d.wrapped.Open(name)
	if err != nil {
		return nil, err
	}
	return &chaosConn{
		wrapped: c,
		eng:     d.eng,
	}, nil
}

type chaosConn struct {
	wrapped driver.Conn
	eng     *engine.Engine
}

func (c *chaosConn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.wrapped.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &chaosStmt{wrapped: s, eng: c.eng, query: query}, nil
}

func (c *chaosConn) Close() error {
	return c.wrapped.Close()
}

func (c *chaosConn) Begin() (driver.Tx, error) {
	return c.wrapped.Begin() //nolint:staticcheck // required by driver.Conn interface; wrapped driver may not implement ConnBeginTx
}

func (c *chaosConn) Ping(ctx context.Context) error {
	if p, ok := c.wrapped.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (c *chaosConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	ec, ok := c.wrapped.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip // let the Stmt path own chaos for this op
	}
	action, err := c.runChaos(ctx, query, classifySQL(query))
	if err != nil {
		return nil, translate(err)
	}
	res, eerr := ec.ExecContext(ctx, query, args)
	reportOutcome(ctx, action, eerr)
	return res, eerr
}

func (c *chaosConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	qc, ok := c.wrapped.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	action, err := c.runChaos(ctx, query, classifySQL(query))
	if err != nil {
		return nil, translate(err)
	}
	rows, qerr := qc.QueryContext(ctx, query, args)
	reportOutcome(ctx, action, qerr)
	return rows, qerr
}

func (c *chaosConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if btc, ok := c.wrapped.(driver.ConnBeginTx); ok {
		return btc.BeginTx(ctx, opts)
	}
	return c.wrapped.Begin() //nolint:staticcheck // -
}

func (c *chaosConn) runChaos(ctx context.Context, query string, method string) (engine.Action, error) {
	if !c.eng.Enabled() {
		return nil, nil
	}
	op := engine.Op{
		Kind:   engine.OpSQL,
		Name:   method,
		Method: method,
		Attrs:  map[string]string{},
	}
	if query != "" {
		op.Attrs["query"] = query
	}
	action := c.eng.Eval(ctx, op)
	return action, action.Before(ctx)
}

type chaosStmt struct {
	wrapped driver.Stmt
	eng     *engine.Engine
	query   string
}

func (s *chaosStmt) Close() error {
	return s.wrapped.Close()
}

func (s *chaosStmt) NumInput() int {
	return s.wrapped.NumInput()
}

func (s *chaosStmt) Exec(args []driver.Value) (driver.Result, error) {
	action, err := s.runChaos(context.Background())
	if err != nil {
		return nil, translate(err)
	}
	res, eerr := s.wrapped.Exec(args) //nolint:staticcheck // required by driver.Stmt interface; delegates to wrapped stmt
	reportOutcome(context.Background(), action, eerr)
	return res, eerr
}

func (s *chaosStmt) Query(args []driver.Value) (driver.Rows, error) {
	action, err := s.runChaos(context.Background())
	if err != nil {
		return nil, translate(err)
	}
	rows, qerr := s.wrapped.Query(args) //nolint:staticcheck // required by driver.Stmt interface; delegates to wrapped stmt
	reportOutcome(context.Background(), action, qerr)
	return rows, qerr
}

func (s *chaosStmt) runChaos(ctx context.Context) (engine.Action, error) {
	if !s.eng.Enabled() {
		return nil, nil
	}
	op := engine.Op{
		Kind:   engine.OpSQL,
		Name:   classifySQL(s.query),
		Method: classifySQL(s.query),
		Attrs:  map[string]string{"query": s.query},
	}
	action := s.eng.Eval(ctx, op)
	return action, action.Before(ctx)
}

// translate converts a fault error into a database/sql-friendly error.
// ErrConnDrop becomes driver.ErrBadConn so database/sql retries.
func translate(err error) error {
	if errors.Is(err, fault.ErrConnDrop) {
		return driver.ErrBadConn
	}
	return err
}

// reportOutcome forwards the wrapped call's error to the engine if the action
// supports it. A nil action (chaos disabled) is a no-op.
func reportOutcome(ctx context.Context, action engine.Action, callErr error) {
	if o, ok := action.(engine.OutcomeReporter); ok {
		o.Outcome(ctx, callErr)
	}
}
