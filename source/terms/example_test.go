package terms_test

import (
	"fmt"

	"github.com/ag4r/chaotic/source/terms"
)

func ExampleCompile() {
	rules, err := terms.Compile(`flaky: kind(http_client),name(/users/*)=2*latency(200ms)`)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(rules), rules[0].Name())
	// Output: 1 flaky
}
