package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	chaossql "github.com/ag4r/chaotic/adapter/sql"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
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
