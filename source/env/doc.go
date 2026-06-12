// Package env builds a RuleSet from an environment variable holding a source/terms
// string, so an already-built binary can be faulted at process start with no
// recompile. It NEVER reads the environment in init(): the caller decides when and
// whether, keeping production opt-in. Pairs with engine.WithProductionGuard.
package env
