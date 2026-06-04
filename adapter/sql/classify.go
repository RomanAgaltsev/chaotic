package sql

import (
	"github.com/ag4r/chaotic/internal/sqlclass"
)

// classifySQL returns the first SQL verb in q, uppercased. Empty for
// whitespace-only input. Used as the Op.Name for SQL chaos rules.
// Users who want finer-grained matching should use MatchAttr or MatchPredicate.
func classifySQL(q string) string {
	return sqlclass.Classify(q).Verb
}
