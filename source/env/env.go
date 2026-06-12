package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

// FromEnv reads the environment variable named varName (or "CHAOTIC_RULES" when
// varName is ""), parses its value with source/terms, and returns a RuleSet. An
// empty or unset variable yields an empty RuleSet, so the engine stays a no-op.
func FromEnv(varName string) (engine.RuleSet, error) {
	if varName == "" {
		varName = "CHAOTIC_RULES"
	}
	s := os.Getenv(varName)
	if strings.TrimSpace(s) == "" {
		return engine.NewRuleSet(nil), nil
	}
	rules, err := terms.Compile(s)
	if err != nil {
		return nil, fmt.Errorf("env %s: %w", varName, err)
	}
	return engine.NewRuleSet(rules), nil
}
