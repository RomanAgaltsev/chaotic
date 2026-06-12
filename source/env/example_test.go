package env_test

import (
	"fmt"
	"os"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/env"
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
