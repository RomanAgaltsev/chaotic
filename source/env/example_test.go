package env_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/env"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

func ExampleFromEnv() {
	os.Setenv("CHAOTIC_RULES", `kind(http_client)=error("boom")`)
	defer os.Unsetenv("CHAOTIC_RULES")

	rs, err := env.FromEnv("CHAOTIC_RULES")
	if err != nil {
		panic(err)
	}
	// Pair with a production guard so a real binary stays opt-in.
	eng := engine.New(engine.WithRuleSource(rs))
	fmt.Println(eng.Enabled())
	// Output: true
}

func ExampleFromEnv_lint() {
	os.Setenv("CHAOTIC_RULES", `wipeout: panic("boom")`)
	defer os.Unsetenv("CHAOTIC_RULES")

	_, err := env.FromEnv("", terms.WithLint(engine.LintReject))
	fmt.Println(errors.Is(err, engine.ErrLintRejected))
	// Output: true
}
