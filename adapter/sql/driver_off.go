//go:build chaos_off

// Package sql (chaos_off build): Register aliases the wrapped driver directly,
// inserting no chaos layer.
package sql

import (
	"database/sql"
	"fmt"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Register, under chaos_off, registers driverName as a direct alias of the
// already-registered wrappedDriverName. The engine is ignored.
func Register(driverName, wrappedDriverName string, eng *engine.Engine) {
	db, err := sql.Open(wrappedDriverName, "")
	if err != nil {
		panic(fmt.Sprintf("chaotic: unknown wrapped driver %q", wrappedDriverName))
	}
	d := db.Driver()
	_ = db.Close()
	sql.Register(driverName, d)
}
