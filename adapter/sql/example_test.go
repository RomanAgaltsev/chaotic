//go:build !chaos_off

package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	chaossql "github.com/RomanAgaltsev/chaotic/adapter/sql"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

func init() { sql.Register("example-fake", fakeDriver{}) }

func ExampleRegister() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpSQL),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("deadlock detected"))),
	).Named("db-flap"))

	// Wrap the already-registered "example-fake" driver with chaos.
	chaossql.Register("chaos:example-fake", "example-fake", eng)
	db, err := sql.Open("chaos:example-fake", "")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	_, err1 := db.ExecContext(context.Background(), "UPDATE accounts SET balance = balance - 1")
	fmt.Println("exec 1 error:", err1)

	_, err2 := db.ExecContext(context.Background(), "UPDATE accounts SET balance = balance - 1")
	fmt.Println("exec 2 error:", err2)
	// Output:
	// exec 1 error: deadlock detected
	// exec 2 error: <nil>
}
