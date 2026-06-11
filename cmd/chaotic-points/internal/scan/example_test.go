package scan_test

import (
	"fmt"

	"github.com/ag4r/chaotic/cmd/chaotic-points/internal/scan"
	"github.com/ag4r/chaotic/engine"
)

func ExampleGate() {
	points := []scan.Point{{Name: "checkout.afterCommit"}}
	specs := []engine.RuleSpec{
		{Name: "typo", Kinds: []string{"explicit"}, NameGlob: "checkout.afterCommt"},
	}
	for _, f := range scan.Gate(points, specs, false) {
		fmt.Printf("%s: %s %q\n", f.Level, f.Message, f.Name)
	}
	// Output:
	// error: unknown explicit point "checkout.afterCommt"
}
