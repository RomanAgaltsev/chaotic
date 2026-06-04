// Package sqlclass classifies SQL strings into a leading verb and a best-effort
// primary table name. It is deliberately small: a hand-rolled scanner, no
// dependency on a full SQL parser. Output drives the Op.Name field used by the
// adapter/sql and adapter/pgx chaos adapters.
package sqlclass

import (
	"strings"
	"unicode"
)

// Classification carries the verb and table extracted from a SQL string.
// Both fields are uppercased ASCII for the verb and case-preserved for the
// table identifier (with surrounding quotes/brackets stripped and a schema
// prefix removed).
type Classification struct {
	Verb  string
	Table string
}

// Classify returns a Classification for sql. Unrecognized input yields the
// leading token as Verb (uppercased) and an empty Table. Empty input yields
// the zero Classification.
func Classify(sql string) Classification {
	s := stripLeadingComments(sql)
	verb, rest := firstToken(s)
	c := Classification{Verb: strings.ToUpper(verb)}
	if c.Verb == "" {
		return c
	}
	c.Table = extractTable(c.Verb, rest)
	return c
}

func stripLeadingComments(s string) string {
	for {
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
		switch {
		case strings.HasPrefix(s, "--"):
			if i := strings.IndexByte(s, '\n'); i >= 0 {
				s = s[i+1:]
				continue
			}
			return ""
		case strings.HasPrefix(s, "/*"):
			if i := strings.Index(s, "*/"); i >= 0 {
				s = s[i+2:]
				continue
			}
			return ""
		}
		return s
	}
}

func firstToken(s string) (token, rest string) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	end := 0
	for end < len(s) && !unicode.IsSpace(rune(s[end])) && s[end] != ';' {
		end++
	}
	return s[:end], strings.TrimLeftFunc(s[end:], unicode.IsSpace)
}

func extractTable(verb, rest string) string {
	switch verb {
	case "SELECT", "DELETE":
		// Look for FROM <table>.
		return tokenAfter(rest, "FROM")
	case "INSERT":
		// INSERT INTO <table>
		return tokenAfter(rest, "INTO")
	case "UPDATE":
		// UPDATE <table>
		t, _ := firstToken(rest)
		return cleanIdent(t)
	case "MERGE":
		// MERGE INTO <table>
		return tokenAfter(rest, "INTO")
	default:
		return ""
	}
}

// tokenAfter returns the next token after the first whole-word, case-insensitive
// occurrence of keyword in s. Returns "" if not found. Both sides of the match
// must be word boundaries, so a column whose name merely contains the keyword
// (e.g. "fromage" containing "FROM") is skipped rather than blanking the result.
func tokenAfter(s, keyword string) string {
	up := strings.ToUpper(s)
	for from := 0; ; {
		rel := strings.Index(up[from:], keyword)
		if rel < 0 {
			return ""
		}
		i := from + rel
		end := i + len(keyword)
		beforeOK := i == 0 || !isIdentChar(rune(up[i-1]))
		afterOK := end >= len(up) || !isIdentChar(rune(up[end]))
		if beforeOK && afterOK {
			t, _ := firstToken(s[end:])
			return cleanIdent(t)
		}
		from = end
	}
}

// cleanIdent strips surrounding quotes/backticks/brackets and a schema prefix.
// "public"."users" → users. `events` → events. [events] → events.
func cleanIdent(t string) string {
	t = strings.TrimSuffix(t, ",")
	t = strings.TrimSuffix(t, ";")
	t = strings.TrimSuffix(t, "(")
	// Strip wrapping quote/bracket characters from each segment.
	parts := splitOnDot(t)
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	return unquoteIdent(last)
}

func splitOnDot(t string) []string {
	// Splits on '.', respecting quoted segments.
	out := []string{}
	cur := strings.Builder{}
	inQuote := byte(0)
	for i := 0; i < len(t); i++ {
		ch := t[i]
		switch {
		case inQuote != 0:
			if ch == inQuote {
				inQuote = 0
			}
			cur.WriteByte(ch)
		case ch == '"' || ch == '`' || ch == '[':
			if ch == '[' {
				inQuote = ']'
			} else {
				inQuote = ch
			}
			cur.WriteByte(ch)
		case ch == '.':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(ch)
		}
	}
	out = append(out, cur.String())
	return out
}

func unquoteIdent(t string) string {
	if len(t) >= 2 {
		switch {
		case t[0] == '"' && t[len(t)-1] == '"',
			t[0] == '`' && t[len(t)-1] == '`':
			return t[1 : len(t)-1]
		case t[0] == '[' && t[len(t)-1] == ']':
			return t[1 : len(t)-1]
		}
	}
	return t
}

func isIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
