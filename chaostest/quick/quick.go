// Package quick provides opinionated one-line shortcuts for the most common
// chaos test setups. Users who outgrow them drop down to engine.NewRule.
package quick

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// FailFirst adds a rule that returns err on the first n matches of kind,
// then becomes inert. The rule named "quick:fail-first" plus a unique
// counter so multiple calls in one test don't collide.
func FailFirst(t testing.TB, eng *engine.Engine, n int, kind engine.Kind, err error) {
	t.Helper()
	eng.AddRule(engine.NewRule(
		engine.MatchKind(kind),
		engine.Times(n),
		engine.WithFault(fault.Error(err)),
	).Named(uniqueName("quick:fail-first")))
}

// SlowAlways adds a rule that delays every matching call by  d.
func SlowAlways(t testing.TB, eng *engine.Engine, kind engine.Kind, d time.Duration) {
	t.Helper()
	eng.AddRule(engine.NewRule(
		engine.MatchKind(kind),
		engine.WithFault(fault.Latency(d)),
	).Named(uniqueName("quick:slow-always")))
}

// PanicOnce adds a rule that panics with v on the first matching call.
func PanicOnce(t testing.TB, eng *engine.Engine, kind engine.Kind, v any) {
	t.Helper()
	eng.AddRule(engine.NewRule(
		engine.MatchKind(kind),
		engine.Times(1),
		engine.WithFault(fault.Panic(v)),
	).Named(uniqueName("quick:panic-once")))
}

var nameCounter atomic.Uint64

// uniqueName ensures distinct names for distinct calls. Resolution is per
// process. Tests using t.Parallel will still get unique names.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s:%d", prefix, nameCounter.Add(1))
}
