package pgx

import (
	"github.com/ag4r/chaotic/engine"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WrapPool returns a *Pool that proxies *pgxpool.Pool through the chaotic
// engine. A nil engine is a programmer error and panics at wrap time.
func WrapPool(p *pgxpool.Pool, eng *engine.Engine) *Pool {
	if eng == nil {
		panic("adapter/pgx: WrapPool requires a non-nil *engine.Engine")
	}
	return &Pool{
		b:   p,
		eng: eng,
		raw: p,
	}
}

// WrapConn returns a *Conn that proxies a standalone *pgx.Conn through the
// chaotic engine. A nil engine is a programmer error and panics at wrap time.
func WrapConn(c *pgxv5.Conn, eng *engine.Engine) *Conn {
	if eng == nil {
		panic("adapter/pgx: WrapConn requires a non-nil *engine.Engine")
	}
	return &Conn{
		b:   &standaloneConnBackend{Conn: c},
		eng: eng,
		raw: c,
	}
}
