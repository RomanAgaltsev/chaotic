package chaostest

import (
	"fmt"
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

// Enable parses a terms string (the source/terms grammar), adds the resulting
// rules to eng, and registers a t.Cleanup that resets the engine when the test
// ends. It returns the rule names in declared order for use with AssertHits.
// Rules the terms string does not name get a deterministic "terms#<i>" name.
// A terms string that fails to compile fails the test via t.Fatalf.
//
// There is no Disable: the registered t.Cleanup resets the engine automatically,
// matching the per-test fresh-engine pattern of New.
func Enable(t testing.TB, eng *engine.Engine, termsStr string) []string {
	t.Helper()
	rules, err := terms.Compile(termsStr)
	if err != nil {
		t.Fatalf("chaostest: invalid terms %q: %v", termsStr, err)
		return nil
	}
	names := make([]string, 0, len(rules))
	for i, r := range rules {
		name := r.Name()
		if name == "" {
			name = fmt.Sprintf("terms#%d", i)
		}
		eng.AddRule(r.Named(name))
		names = append(names, name)
	}
	t.Cleanup(eng.Reset)
	return names
}
