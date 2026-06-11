//go:build !chaos_off

package nats

import (
	"errors"
	"fmt"

	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleWrapConn() {
	// Fail the first publish, then recover.
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNATS),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("server down"))),
	).Named("nats-flap"))

	// In production you wrap a live connection: cc := chaosnats.WrapConn(nc, eng).
	// Here a fake stands in for the server so the example is hermetic.
	c := &Conn{conn: &fakeConn{}, eng: eng}

	publish := func() error {
		return c.Publish("events", []byte("hi"))
	}

	fmt.Println("attempt 1:", publish())
	fmt.Println("attempt 2:", publish())
	// Output:
	// attempt 1: server down
	// attempt 2: <nil>
}
