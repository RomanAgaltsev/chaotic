//go:build !chaos_off

package io_test

import (
	"fmt"
	"io"
	"strings"

	chaosio "github.com/ag4r/chaotic/adapter/io"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/fault"
)

func ExampleWrapReader() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Always(),
		engine.WithFault(fault.Truncate(5)),
	).Named("trunc"))

	r := chaosio.WrapReader(strings.NewReader("hello, world"), eng)
	got, _ := io.ReadAll(r)
	fmt.Printf("%q\n", got)
	// Output: "hello"
}
