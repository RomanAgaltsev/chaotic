//go:build chaos_off

package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	chaossql "github.com/ag4r/chaotic/adapter/sql"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func TestNoChaosUnderChaosOff(t *testing.T) {
	boom := errors.New("boom")
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.WithFault(fault.Error(boom)),
	))
	chaossql.Register("chaos:off", "failing-shim", eng)
	db, err := sql.Open("chaos:off", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, qerr := db.QueryContext(context.Background(), "SELECT 1")
	if errors.Is(qerr, boom) {
		t.Fatal("chaos fired under chaos_off build")
	}
}
