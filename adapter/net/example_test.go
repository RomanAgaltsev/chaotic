//go:build !chaos_off

package net_test

import (
	"errors"
	"fmt"
	"net"

	chaosnet "github.com/ag4r/chaotic/adapter/net"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleWrapConn() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpNet),
		engine.Times(1),
		engine.WithFault(fault.Error(errors.New("link down"))),
	).Named("net-flap"))

	a, b := net.Pipe()
	defer func() { _ = a.Close(); _ = b.Close() }()
	c := chaosnet.WrapConn(a, eng)

	// First read faults; a real reader would retry/reconnect.
	_, err := c.Read(make([]byte, 4))
	fmt.Println("read 1:", err)
	// Output: read 1: link down
}
