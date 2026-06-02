// Package file loads chaotic rules from a YAML document (one file = one rule
// set). Use Load/Parse for a one-shot read, or Watch for live reloads.
package file

import (
	"fmt"
	"os"

	"github.com/ag4r/chaotic/engine"
	"gopkg.in/yaml.v3"
)

// Document is the YAML schemaL a meta block and a list of rule specs.
type Document struct {
	Meta  Meta              `yaml:"meta"`
	Rules []engine.RuleSpec `yaml:"rules"`
}

// Meta cerries schema metadata. Version is reserved for future migration.
type Meta struct {
	Version int `yaml:"version"`
}

// Load reads and parses path into a RuleSet.
func Load(path string) (engine.RuleSet, error) {
	data, err := os.ReadFile(path) //nolint:gosec // -
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse builds a validated RuleSet from YAML bytes. Rule names must be unique.
func Parse(data []byte) (engine.RuleSet, error) {
	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("chaotic: parse rules: %w", err)
	}
	rules := make([]engine.Rule, 0, len(doc.Rules))
	seen := map[string]bool{}
	for i, spec := range doc.Rules {
		if spec.Name != "" {
			if seen[spec.Name] {
				return nil, fmt.Errorf("chaotic: duplicate rule name %q", spec.Name)
			}
			seen[spec.Name] = true
		}
		r, err := engine.BuildRule(spec)
		if err != nil {
			return nil, fmt.Errorf("chaotic: rule %d (%q): %w", i, spec.Name, err)
		}
		rules = append(rules, r)
	}
	return engine.NewRuleSet(rules), nil
}
