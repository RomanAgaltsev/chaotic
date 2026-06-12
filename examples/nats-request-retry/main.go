// Command nats-request-retry demonstrates a request/reply caller that retries
// through a transient NATS outage, and proves the retry works using the chaotic
// adapter/nats wrapper to fail requests on demand — no real outage, no flaky timing.
package main

import (
	"context"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"

	chaosnats "github.com/RomanAgaltsev/chaotic/adapter/nats"
)

// RequestWithRetry sends a request, retrying up to attempts times with a short
// linear backoff, so a transient outage does not surface as a hard error.
func RequestWithRetry(cc *chaosnats.Conn, subj string, data []byte, timeout time.Duration, attempts int) (*natsgo.Msg, error) {
	var msg *natsgo.Msg
	var err error
	for i := range attempts {
		msg, err = cc.Request(subj, data, timeout)
		if err == nil {
			return msg, nil
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	return msg, err
}

func main() {
	_ = context.Background()
	fmt.Println("run `go test` in this directory to see request-retry survive a chaos outage")
}
