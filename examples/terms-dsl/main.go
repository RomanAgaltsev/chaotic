// Command terms-dsl demonstrates activating chaos from a one-line terms string:
// no Go rule-building code, just terms.Compile + AddRule, driving an explicit
// chaos.Point.
package main

import (
	"context"
	"fmt"

	"github.com/ag4r/chaotic/chaos"
	"github.com/ag4r/chaotic/engine"
	"github.com/ag4r/chaotic/source/terms"
)

// Activate compiles a terms string and installs the rules on eng.
func Activate(eng *engine.Engine, spec string) error {
	rules, err := terms.Compile(spec)
	if err != nil {
		return err
	}
	for _, r := range rules {
		eng.AddRule(r)
	}
	return nil
}

func main() {
	fmt.Println("run `go test` in this directory to see a one-liner activate chaos")
	_ = context.Background
	_ = chaos.Point
}
