package sql

import (
	"strings"
	"unicode"
)

// classifySQL returns the first SQL verb in q, uppercased. Empty for
// whitespace-only input. Used as the Op.Name for SQL chaos rules.
// Users who want finer-grained matching should use MatchAttr or MatchPredicate.
func classifySQL(q string) string {
	start := 0
	for start < len(q) && unicode.IsSpace(rune(q[start])) {
		start++
	}
	end := start
	for end < len(q) && !unicode.IsSpace(rune(q[end])) && q[end] != ';' && q[end] != '(' {
		end++
	}
	return strings.ToUpper(q[start:end])
}
