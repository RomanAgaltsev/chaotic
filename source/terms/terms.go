package terms

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ag4r/chaotic/engine"
)

// splitTop splits s on sep at parenthesis depth 0, so separators inside (...)
// — e.g. the comma in jitter(1ms,2ms) — are not split points.
func splitTop(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// indexTop returns the index of the first sep at parenthesis depth 0, or -1.
func indexTop(s string, sep byte) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case sep:
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitCall parses "ident(args)" into ("ident", "args") or a bare "ident" into
// ("ident", ""). It errors if there is a "(" without a closing ")".
func splitCall(s string) (name, args string, err error) {
	open := strings.IndexByte(s, '(')
	if open < 0 {
		return s, "", nil
	}
	if !strings.HasSuffix(s, ")") {
		return "", "", fmt.Errorf("terms: %q missing closing )", s)
	}
	return s[:open], s[open+1 : len(s)-1], nil
}

// Parse turns a terms string into RuleSpecs (the declarative form), so the same
// BuildRule validation and the LintSpecs blast-radius check apply. Rules are
// separated by ';'. See the package doc for the grammar.
func Parse(s string) ([]engine.RuleSpec, error) {
	var specs []engine.RuleSpec
	for _, raw := range splitTop(s, ';') {
		rule := strings.TrimSpace(raw)
		if rule == "" {
			continue
		}
		spec, err := parseRule(rule)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("terms: empty ruleset")
	}
	return specs, nil
}

func parseRule(s string) (engine.RuleSpec, error) {
	var spec engine.RuleSpec
	if strings.Contains(s, "->") {
		return spec, fmt.Errorf("terms: %q: chained terms (\"->\") are not supported yet "+
			"(staged faults pending, roadmap §10); use one term per rule", s)
	}
	// Optional "name:" prefix (top-level colon, before any '=').
	if i := indexTop(s, ':'); i >= 0 {
		spec.Name = strings.TrimSpace(s[:i])
		s = strings.TrimSpace(s[i+1:])
	}
	// Optional "selectors=" prefix (top-level '=').
	term := s
	if i := indexTop(s, '='); i >= 0 {
		sel := strings.TrimSpace(s[:i])
		if sel == "" {
			return spec, fmt.Errorf("terms: %q has an empty selector list before '='", s)
		}
		term = strings.TrimSpace(s[i+1:])
		if err := parseSelectors(sel, &spec); err != nil {
			return spec, err
		}
	}
	if err := parseTerm(term, &spec); err != nil {
		return spec, err
	}
	return spec, nil
}

func parseSelectors(s string, spec *engine.RuleSpec) error {
	for _, raw := range splitTop(s, ',') {
		sel := strings.TrimSpace(raw)
		if sel == "" {
			continue
		}
		name, args, err := splitCall(sel)
		if err != nil {
			return err
		}
		switch name {
		case "kind":
			spec.Kinds = append(spec.Kinds, strings.TrimSpace(args))
		case "name":
			spec.NameGlob = strings.TrimSpace(args)
		case "attr":
			k, v, ok := strings.Cut(args, "=")
			if !ok {
				return fmt.Errorf("terms: attr selector %q must be attr(key=value)", sel)
			}
			if spec.Attrs == nil {
				spec.Attrs = map[string]string{}
			}
			spec.Attrs[strings.TrimSpace(k)] = strings.TrimSpace(v)
		default:
			return fmt.Errorf("terms: unknown selector %q (want kind/name/attr)", name)
		}
	}
	return nil
}

func parseTerm(s string, spec *engine.RuleSpec) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("terms: empty term")
	}
	// Mode prefix (int* or float%) is searched only before the first '(' so a
	// '%' inside an arg string (e.g. error("50%")) is not mistaken for a mode.
	head := s
	if p := strings.IndexByte(s, '('); p >= 0 {
		head = s[:p]
	}
	if i := strings.IndexAny(head, "*%"); i >= 0 {
		switch s[i] {
		case '*':
			n, err := strconv.Atoi(strings.TrimSpace(s[:i]))
			if err != nil {
				return fmt.Errorf("terms: bad count mode %q: %w", s[:i], err)
			}
			spec.Counter = engine.CounterSpec{Type: "times", N: n}
		case '%':
			f, err := strconv.ParseFloat(strings.TrimSpace(s[:i]), 64)
			if err != nil {
				return fmt.Errorf("terms: bad percent mode %q: %w", s[:i], err)
			}
			spec.Counter = engine.CounterSpec{Type: "probability", P: f / 100}
		}
		s = strings.TrimSpace(s[i+1:])
	}
	name, args, err := splitCall(s)
	if err != nil {
		return err
	}
	return parseAction(strings.TrimSpace(name), args, spec)
}

func parseAction(name, args string, spec *engine.RuleSpec) error {
	switch name {
	case "latency":
		spec.Faults = append(spec.Faults, engine.FaultSpec{Type: "latency", Duration: strings.TrimSpace(args)})
	case "jitter":
		lo, hi, ok := strings.Cut(args, ",")
		if !ok {
			return fmt.Errorf("terms: jitter needs jitter(min,max), got %q", args)
		}
		spec.Faults = append(spec.Faults, engine.FaultSpec{Type: "jittered", Min: strings.TrimSpace(lo), Max: strings.TrimSpace(hi)})
	case "error":
		msg, err := unquote(args)
		if err != nil {
			return fmt.Errorf("terms: error message: %w", err)
		}
		spec.Faults = append(spec.Faults, engine.FaultSpec{Type: "error", Message: msg})
	case "panic":
		v, err := unquote(args)
		if err != nil {
			return fmt.Errorf("terms: panic value: %w", err)
		}
		spec.Faults = append(spec.Faults, engine.FaultSpec{Type: "panic", Value: v})
	case "conndrop":
		spec.Faults = append(spec.Faults, engine.FaultSpec{Type: "conn_drop"})
	case "off":
		// Rule present but inert: no faults.
	default:
		return fmt.Errorf("terms: unknown action %q (want latency/jitter/error/panic/conndrop/off)", name)
	}
	return nil
}

// unquote strips surrounding double quotes (Go-quoted) if present, else returns
// the trimmed input verbatim.
func unquote(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' {
		return strconv.Unquote(s)
	}
	return s, nil
}

// Compile is Parse followed by engine.BuildRule for each spec — the convenience
// path when you want rules to AddRule directly. Validation (unknown kinds, bad
// durations, out-of-range probabilities) is performed by BuildRule, so a
// structurally valid terms string can still fail here with a clear error.
func Compile(s string) ([]engine.Rule, error) {
	specs, err := Parse(s)
	if err != nil {
		return nil, err
	}
	rules := make([]engine.Rule, 0, len(specs))
	for _, sp := range specs {
		r, err := engine.BuildRule(sp)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}
