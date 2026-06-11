// Package property is a property-testing harness for chaos rules: it runs a test
// body against many randomized rule configurations and, on the first failure,
// minimizes to the smallest rule set that still triggers it.
//
// Provide one RuleGen per dimension of the search space and a body that returns a
// non-nil error when its invariant is violated:
//
//	property.Test(t,
//	    []property.RuleGen{
//	        func(r *rand.Rand) engine.Rule {
//	            return engine.NewRule(engine.MatchKind(engine.OpHTTPClient),
//	                engine.Probability(r.Float64(), int64(r.Uint64())),
//	                engine.WithFault(fault.ConnDrop())).Named("net")
//	        },
//	    },
//	    func(eng *engine.Engine) error {
//	        // exercise the system under test against eng; return an error if an
//	        // invariant breaks for this configuration.
//	        return nil
//	    },
//	)
//
// On failure Test reports the seed and the minimal failing generator indices, and
// a one-line recipe to replay that single configuration (WithSeed + WithRuns(1)).
// Default exploration is 100 runs, overridable with -chaos-property-runs or
// property.WithRuns.
package property
