package engine

import (
	"context"
	"time"
)

// MatchTimeWindow matches Ops only when the local wall clock falls within the
// daily window [startH:startM, endH:endM). Hours are 0–23, minutes 0–59, in the
// process's local time zone. Useful for scheduled chaos, e.g. "only inject
// between 02:00 and 04:00 during the canary window". A window where start == end
// matches nothing; a window spanning midnight (start > end) is supported.
func MatchTimeWindow(startH, startM, endH, endM int) RuleOption {
	return matchTimeWindowAt(startH, startM, endH, endM, time.Now)
}

// matchTimeWindowAt is MatchTimeWindow with an injectable clock, for tests.
func matchTimeWindowAt(startH, startM, endH, endM int, now func() time.Time) RuleOption {
	start := startH*60 + startM
	end := endH*60 + endM
	return func(r *Rule) {
		r.matchers = append(r.matchers, func(_ context.Context, _ Op) bool {
			t := now()
			cur := t.Hour()*60 + t.Minute()
			if start == end {
				return false
			}
			if start < end {
				return cur >= start && cur < end
			}
			// Window wraps past midnight (e.g. 22:00–02:00).
			return cur >= start || cur < end
		})
	}
}
