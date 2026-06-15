package engine

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/RomanAgaltsev/chaotic/fault"
)

// lintLatencyCeiling is the latency above which LintSpecs warns: a single
// injected sleep this long is rarely an intentional, safe chaos experiment.
const lintLatencyCeiling = 5 * time.Second

// Severity ranks a lint Finding. Only SeverityHigh findings fail Report.OK.
type Severity int

// Severity levels in ascending order of seriousness.
const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityHigh
)

// String returns the lowercase severity name.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarn:
		return "warn"
	case SeverityHigh:
		return "high"
	}
	return "unknown"
}

// Finding is one blast-radius hazard the linter detected. Rule is the offending
// rule's name (or "<unnamed>" when it has none).
type Finding struct {
	Severity Severity
	Rule     string
	Message  string
}

// Report is the result of a lint pass.
type Report struct {
	Findings []Finding
}

// OK reports whether the report contains no SeverityHigh findings. Authoring
// tools can gate on this to fail a build on a high-severity hazard while
// tolerating warnings.
func (r Report) OK() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityHigh {
			return false
		}
	}
	return true
}

func lintName(name string) string {
	if name == "" {
		return "<unnamed>"
	}
	return name
}

// isTerminalFaultType reports whether a fault spec type permanently fails the
// call (a panic or a connection drop) rather than merely delaying it. Kept in
// sync with terminalKindName so the spec-level and introspection-level linters
// agree on what counts as terminal.
func isTerminalFaultType(t string) bool {
	return t == "panic" || t == "conn_drop" || t == "disconnect"
}

// terminalKindName returns a human-readable name for a terminal fault kind
// (panic or connection drop), or "" if k is not terminal.
func terminalKindName(k fault.Kind) string {
	switch k {
	case fault.KindPanic:
		return "panic"
	case fault.KindConnDrop:
		return "conn_drop"
	case fault.KindDisconnect:
		return "disconnect"
	}
	return ""
}

// Lint inspects programmatic rules via RuleInfo. It is coarse: closures hide
// globs and probability values, so it flags only structural hazards visible
// through introspection — chiefly a rule that matches every operation on every
// call (no matchers, Always counter), which is far riskier when its faults are
// terminal (panic or connection drop).
func Lint(rules []Rule) Report {
	var rep Report
	for _, r := range rules {
		info := r.Info()
		if !info.Unconstrained || info.Counter != CounterAlways {
			continue
		}
		var terminal []string
		for _, k := range info.Faults {
			if name := terminalKindName(k); name != "" {
				terminal = append(terminal, name)
			}
		}
		if len(terminal) > 0 {
			rep.Findings = append(rep.Findings, Finding{
				Severity: SeverityHigh,
				Rule:     lintName(info.Name),
				Message: fmt.Sprintf(
					"matches every operation on every call and injects %s",
					strings.Join(terminal, ", ")),
			})
			continue
		}
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityWarn,
			Rule:     lintName(info.Name),
			Message:  "matches every operation on every call (no matchers, Always counter)",
		})
	}
	return rep
}

// LintSpecs inspects declarative specs and can see globs, probabilities, and
// durations the programmatic Lint cannot. It is the richer analog: it flags a
// wildcard name glob, a probability that always fires, latency above
// lintLatencyCeiling, a terminal fault on a globally-scoped spec, and two specs
// that target the same kind+glob (an overlap whose combined effect is easy to
// underestimate).
func LintSpecs(specs []RuleSpec) Report {
	var rep Report

	for _, s := range specs {
		name := lintName(s.Name)

		if s.NameGlob == "*" {
			rep.Findings = append(rep.Findings, Finding{
				Severity: SeverityWarn,
				Rule:     name,
				Message:  `name glob "*" matches every operation name`,
			})
		}

		if s.Counter.Type == "probability" && s.Counter.P >= 1.0 {
			rep.Findings = append(rep.Findings, Finding{
				Severity: SeverityWarn,
				Rule:     name,
				Message:  "probability >= 1.0 always fires; use the Always counter instead",
			})
		}

		broad := len(s.Kinds) == 0 && (s.NameGlob == "" || s.NameGlob == "*")
		for _, fs := range s.Faults {
			if (fs.Type == "slow_reader" || fs.Type == "slow_writer") && fs.Rate == 0 {
				rep.Findings = append(rep.Findings, Finding{
					Severity: SeverityWarn,
					Rule:     name,
					Message:  fmt.Sprintf("%s rate 0 blocks until context cancellation - a stream that never ends", fs.Type),
				})
			}
			lintFault(name, broad, fs, &rep)
		}
		for _, st := range s.Stages {
			for _, fs := range st.Faults {
				lintFault(name, broad, fs, &rep)
			}
		}
		if len(s.Stages) > 0 {
			last := s.Stages[len(s.Stages)-1]
			if last.Times == 0 {
				for _, fs := range last.Faults {
					if isTerminalFaultType(fs.Type) {
						rep.Findings = append(rep.Findings, Finding{
							Severity: SeverityWarn,
							Rule:     name,
							Message: fmt.Sprintf(
								"staged rule fails permanently after its transient stages (open-ended final stage injects %s)", fs.Type),
						})
					}
				}
			}
		}
	}

	rep.Findings = append(rep.Findings, lintOverlaps(specs)...)
	return rep
}

// specLatency returns the sleep duration a latency or jittered fault spec
// injects (the max for jittered), and whether the spec is such a fault with a
// parseable duration.
func specLatency(fs FaultSpec) (time.Duration, bool) {
	switch fs.Type {
	case "latency":
		d, err := time.ParseDuration(fs.Duration)
		return d, err == nil
	case "jittered":
		d, err := time.ParseDuration(fs.Max)
		return d, err == nil
	}
	return 0, false
}

// lintOverlaps flags each spec that targets the same kind+glob as an earlier
// spec. Overlapping rules compound: their effects stack on the same operations.
func lintOverlaps(specs []RuleSpec) []Finding {
	var findings []Finding
	first := make(map[string]string) // signature -> first spec's name
	for _, s := range specs {
		sig := specSignature(s)
		if prior, seen := first[sig]; seen {
			findings = append(findings, Finding{
				Severity: SeverityWarn,
				Rule:     lintName(s.Name),
				Message:  fmt.Sprintf("targets the same kind+glob as rule %q", prior),
			})
			continue
		}
		first[sig] = lintName(s.Name)
	}
	return findings
}

// specSignature is a stable key for a spec's match scope: its sorted kinds plus
// name glob. Two specs with the same signature target the same operations.
func specSignature(s RuleSpec) string {
	kinds := append([]string(nil), s.Kinds...)
	slices.Sort(kinds)
	return strings.Join(kinds, ",") + "|" + s.NameGlob
}

// lintFault appends the latency-ceiling and terminal-on-broad-scope findings for
// a single fault spec. Shared by the flat-faults and staged-faults passes.
func lintFault(name string, broad bool, fs FaultSpec, rep *Report) {
	if d, ok := specLatency(fs); ok && d > lintLatencyCeiling {
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityWarn,
			Rule:     name,
			Message:  fmt.Sprintf("latency %s exceeds the %s ceiling", d, lintLatencyCeiling),
		})
	}
	if broad && isTerminalFaultType(fs.Type) {
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityHigh,
			Rule:     name,
			Message:  fmt.Sprintf("injects %s with no kind or name scope (matches everything)", fs.Type),
		})
	}
}
