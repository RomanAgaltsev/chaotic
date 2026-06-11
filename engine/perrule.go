package engine

// WithPerRuleRateLimit caps how often THIS rule's faults actually fire to rps per
// second (burst rps), independent of other rules and of the engine-wide
// WithRateLimit. When the rule is throttled it is skipped and evaluation
// continues to the next rule, so a lower-priority rule can still match. Panics if
// rps < 1 (matches WithRateLimit). The cap reuses the engine's token bucket.
func WithPerRuleRateLimit(rps int) RuleOption {
	return func(r *Rule) {
		r.rateLimiter = newTokenBucket(rps)
	}
}
