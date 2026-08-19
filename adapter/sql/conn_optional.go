//go:build !chaos_off

package sql

import (
	"context"
	"database/sql/driver"
)

// This file forwards database/sql's optional Conn and Stmt interfaces through
// the chaos wrapper. Each is safe to implement unconditionally because its
// fallback is exactly what database/sql assumes when the interface is absent:
// no session state to reset, a valid connection, "use the default converter",
// and the context-less Prepare. driver.Pinger is the exception and is handled
// by a separate type in driver.go, because "cannot ping" and "ping succeeded"
// are different answers with no ErrSkip escape between them.
//
// None of these inject faults: they are lifecycle and conversion hooks, not
// operations, so they stay off the engine path entirely.

// ResetSession forwards to the wrapped conn when it is a driver.SessionResetter.
// A conn that is not one has no session state to reset, which is exactly what
// database/sql assumes when the interface is absent.
func (c *chaosConn) ResetSession(ctx context.Context) error {
	if sr, ok := c.wrapped.(driver.SessionResetter); ok {
		return sr.ResetSession(ctx)
	}
	return nil
}

// IsValid forwards to the wrapped conn when it is a driver.Validator. When it is
// not, the conn is reported valid: database/sql discards a conn only on false.
func (c *chaosConn) IsValid() bool {
	if v, ok := c.wrapped.(driver.Validator); ok {
		return v.IsValid()
	}
	return true
}

// CheckNamedValue forwards to the wrapped conn when it is a
// driver.NamedValueChecker. driver.ErrSkip is the documented signal for "use the
// default converter", so it is the correct answer when the wrapped conn does not
// implement the interface. Without this, drivers with custom argument types
// (pq.Array, pgtype) fail argument conversion through the chaos wrapper.
func (c *chaosConn) CheckNamedValue(nv *driver.NamedValue) error {
	if nvc, ok := c.wrapped.(driver.NamedValueChecker); ok {
		return nvc.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}

// PrepareContext forwards to the wrapped conn when it is a
// driver.ConnPrepareContext so the caller's context reaches the driver, and
// falls back to Prepare otherwise — the same fallback database/sql performs.
func (c *chaosConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	pc, ok := c.wrapped.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}
	s, err := pc.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &chaosStmt{wrapped: s, eng: c.eng, query: query}, nil
}

// CheckNamedValue forwards to the wrapped stmt when it is a
// driver.NamedValueChecker. database/sql consults the Stmt before the Conn, so
// the wrapper must carry it at both levels.
func (s *chaosStmt) CheckNamedValue(nv *driver.NamedValue) error {
	if nvc, ok := s.wrapped.(driver.NamedValueChecker); ok {
		return nvc.CheckNamedValue(nv)
	}
	return driver.ErrSkip
}
