package engine

import (
	"context"
	"regexp"
)

// MatchNameRegex matches Ops whose Name matches re. Use it when MatchName's
// path.Match globbing is too constrained — notably, path.Match's * does not cross
// "/", whereas a regexp can. re must be pre-compiled (regexp.MustCompile or a
// checked regexp.Compile) so an invalid pattern surfaces at construction, not at
// fire-time. A nil re matches nothing.
func MatchNameRegex(re *regexp.Regexp) RuleOption {
	return func(r *Rule) {
		r.matchers = append(r.matchers, func(_ context.Context, op Op) bool {
			return re != nil && re.MatchString(op.Name)
		})
	}
}
