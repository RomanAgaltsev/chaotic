// Command slow-body-read demonstrates adapter/io: a consumer reads a response
// body through a chaosio-wrapped reader, and chaos either trickles the body
// slowly or truncates it mid-payload.
package main

import (
	"fmt"
	"io"
	"strings"

	chaosio "github.com/RomanAgaltsev/chaotic/adapter/io"
	"github.com/RomanAgaltsev/chaotic/engine"
	"github.com/RomanAgaltsev/chaotic/fault"
)

// ReadBody reads the whole body from src through the engine. Returns the bytes
// read and any error (io.ErrUnexpectedEOF-style truncation surfaces here).
func ReadBody(eng *engine.Engine, src string) ([]byte, error) {
	r := chaosio.WrapReader(strings.NewReader(src), eng)
	return io.ReadAll(r)
}

func main() {
	eng := engine.New().AddRule(engine.NewRule(
		engine.MatchKind(engine.OpIO),
		engine.Times(1),
		engine.WithFault(fault.Truncate(4)),
	).Named("trunc"))
	got, _ := ReadBody(eng, `{"ok":true}`)
	fmt.Printf("read %q\n", got)
}
