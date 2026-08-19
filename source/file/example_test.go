package file_test

import (
	"errors"
	"fmt"

	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/source/file"
)

func ExampleWithLint() {
	doc := []byte("meta:\n  version: 1\nrules:\n  - name: wipeout\n    faults:\n      - type: panic\n        message: boom\n")

	_, err := file.Parse(doc, file.WithLint(engine.LintReject))
	fmt.Println(errors.Is(err, engine.ErrLintRejected))
	// Output: true
}
