package bench

import (
	"testing"

	"github.com/RomanAgaltsev/chaotic/engine"
)

// Profile names and configures one chaos profile for a benchmark sub-run. Apply
// installs the profile's rules on the engine; a nil Apply is a clean baseline
// (no chaos).
type Profile struct {
	Name  string
	Apply func(*engine.Engine)
}

// Run benchmarks body once per profile, in declared slice order, as named
// sub-benchmarks. Before each profile it Resets eng and applies the profile, so
// profiles do not leak rules into one another. Allocations are reported per
// profile; ns/op comes from the framework. Pipe the output into benchstat to
// compare profiles row-by-row (slice order keeps the rows stable across runs).
func Run(b *testing.B, eng *engine.Engine, profiles []Profile, body func(*testing.B)) {
	b.Helper()
	for _, p := range profiles {
		b.Run(p.Name, func(sub *testing.B) {
			eng.Reset()
			if p.Apply != nil {
				p.Apply(eng)
			}
			sub.ReportAllocs()
			sub.ResetTimer()
			body(sub)
		})
	}
}
