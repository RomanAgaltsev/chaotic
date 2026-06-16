// Command chaotic-points discovers chaos.Point / chaos.PointWith call sites and
// gates a rules config against typo'd explicit-point names. See README.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/RomanAgaltsev/chaotic/cmd/chaotic-points/internal/scan"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "list":
		os.Exit(cmdList(os.Args[2:]))
	case "lint":
		os.Exit(cmdLint(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  chaotic-points list [--json] [-C dir] [packages...]")
	fmt.Fprintln(os.Stderr, "  chaotic-points lint [--rules f.json]... [--terms s]... [--terms-file f]... [--strict] [-C dir] [packages...]")
}

func cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit JSON")
	dir := fs.String("C", "", "scan this module directory instead of the current one")
	_ = fs.Parse(args)
	pts, err := scanPoints(*dir, fs.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "chaotic-points:", err)
		return 2
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(pts)
		return 0
	}
	for _, p := range pts {
		name := p.Name
		if p.Dynamic {
			name = "<dynamic>"
		}
		fmt.Fprintf(os.Stdout, "%s\t%s:%d\n", name, p.File, p.Line)
	}
	return 0
}

// repeatableFlag collects a flag that may appear multiple times.
type repeatableFlag []string

func (r *repeatableFlag) String() string { return strings.Join(*r, ",") }
func (r *repeatableFlag) Set(v string) error {
	*r = append(*r, v)
	return nil
}

func cmdLint(args []string) int {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	var rulesFiles, termsStrings, termsFiles repeatableFlag
	fs.Var(&rulesFiles, "rules", "JSON file of []engine.RuleSpec (repeatable)")
	fs.Var(&termsStrings, "terms", "inline terms-DSL string (repeatable)")
	fs.Var(&termsFiles, "terms-file", "file containing a terms-DSL string (repeatable)")
	strict := fs.Bool("strict", false, "treat glob-matches-nothing as an error")
	dir := fs.String("C", "", "scan this module directory instead of the current one")
	_ = fs.Parse(args)

	if len(rulesFiles)+len(termsStrings)+len(termsFiles) == 0 {
		fmt.Fprintln(os.Stderr, "chaotic-points lint: at least one of --rules/--terms/--terms-file is required")
		return 2
	}

	specs, err := loadSpecs(rulesFiles, termsStrings, termsFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chaotic-points:", err)
		return 2
	}
	pts, err := scanPoints(*dir, fs.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "chaotic-points:", err)
		return 2
	}

	findings := scan.Gate(pts, specs, *strict)
	hasError := false
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "%s: rule %q: %s %q\n", f.Level, f.Rule, f.Message, f.Name)
		if f.Level == "error" {
			hasError = true
		}
	}
	if anyDynamic(pts) {
		fmt.Fprintln(os.Stderr, "note: some points have non-constant names and were not gated")
	}
	if hasError {
		return 1
	}
	return 0
}

func loadSpecs(rulesFiles, termsStrings, termsFiles repeatableFlag) ([]engine.RuleSpec, error) {
	var specs []engine.RuleSpec
	for _, f := range rulesFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var ss []engine.RuleSpec
		if err := json.Unmarshal(b, &ss); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		specs = append(specs, ss...)
	}
	for _, s := range termsStrings {
		ss, err := terms.Parse(s)
		if err != nil {
			return nil, err
		}
		specs = append(specs, ss...)
	}
	for _, f := range termsFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		ss, err := terms.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		specs = append(specs, ss...)
	}
	return specs, nil
}

// scanPoints discovers points in dir (its own module root) when -C is given,
// otherwise in the current module. Needed to scan a nested module such as a
// testdata fixture, which the current module's `go list` cannot reach.
func scanPoints(dir string, patterns []string) ([]scan.Point, error) {
	if dir != "" {
		return scan.Dir(dir, patterns...)
	}
	return scan.Scan(patterns...)
}

func anyDynamic(pts []scan.Point) bool {
	for _, p := range pts {
		if p.Dynamic {
			return true
		}
	}
	return false
}
