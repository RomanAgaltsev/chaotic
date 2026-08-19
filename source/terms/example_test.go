package terms_test

import (
	"errors"
	"fmt"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/terms"
)

func ExampleCompile() {
	rules, err := terms.Compile(`flaky: kind(http_client),name(/users/*)=2*latency(200ms)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(rules), rules[0].Name())
	// Output: 1 flaky
}

func ExampleWithLint() {
	// A terminal fault with no kind and no name scope matches every operation
	// on every call — the hazard LintReject exists to stop.
	_, err := terms.Compile(`wipeout: panic("boom")`, terms.WithLint(engine.LintReject))
	fmt.Println(errors.Is(err, engine.ErrLintRejected))
	// Output: true
}
