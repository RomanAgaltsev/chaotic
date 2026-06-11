package scan

import (
	"path"
	"strings"

	"github.com/ag4r/chaotic/engine"
)

// Finding is one gating problem: a rule references an explicit-point name that
// is not (or may not be) present among the discovered points.
type Finding struct {
	Rule    string // the rule's name (RuleSpec.Name), or "" if unnamed
	Name    string // the offending name_glob
	Level   string // "error" or "warning"
	Message string
}

// Gate checks every explicit-point rule's NameGlob against the discovered points.
// A literal name absent from the points is an error; a glob matching zero points
// is a warning (an error when strict). Rules not targeting the explicit kind, and
// rules with an empty NameGlob, are ignored. Dynamic points contribute no names.
func Gate(points []Point, specs []engine.RuleSpec, strict bool) []Finding {
	known := make(map[string]bool, len(points))
	for _, p := range points {
		if !p.Dynamic {
			known[p.Name] = true
		}
	}
	var fs []Finding
	for _, sp := range specs {
		if !targetsExplicit(sp) || sp.NameGlob == "" {
			continue
		}
		if isGlob(sp.NameGlob) {
			if !anyMatch(sp.NameGlob, known) {
				level := "warning"
				if strict {
					level = "error"
				}
				fs = append(fs, Finding{Rule: sp.Name, Name: sp.NameGlob, Level: level, Message: "glob matches no known explicit point"})
			}
			continue
		}
		if !known[sp.NameGlob] {
			fs = append(fs, Finding{Rule: sp.Name, Name: sp.NameGlob, Level: "error", Message: "unknown explicit point"})
		}
	}
	return fs
}

// targetsExplicit reports whether a spec can match the explicit kind: either it
// lists "explicit", or it lists no kinds (matching all kinds, explicit included).
func targetsExplicit(sp engine.RuleSpec) bool {
	if len(sp.Kinds) == 0 {
		return true
	}
	for _, k := range sp.Kinds {
		if k == "explicit" {
			return true
		}
	}
	return false
}

func isGlob(s string) bool { return strings.ContainsAny(s, "*?[") }

func anyMatch(glob string, known map[string]bool) bool {
	for name := range known {
		if ok, _ := path.Match(glob, name); ok {
			return true
		}
	}
	return false
}
