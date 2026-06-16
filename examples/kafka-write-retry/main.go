// Command kafka-write-retry demonstrates a producer that retries through a
// transient Kafka outage, and proves the retry works using the chaotic
// adapter/kafka wrapper to fail writes on demand — no real outage, no flaky timing.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// writer is the minimal surface the retry helper needs; both *kafka.Writer and
// *chaoskafka.Writer satisfy it.
type writer interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
}

// WriteWithRetry writes msgs, retrying up to attempts times with a short linear
// backoff, so a transient broker outage does not lose the batch.
func WriteWithRetry(ctx context.Context, w writer, attempts int, msgs ...kafkago.Message) error {
	var err error
	for i := range attempts {
		if err = w.WriteMessages(ctx, msgs...); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	return err
}

func main() {
	fmt.Fprintln(os.Stdout, "run `go test` in this directory to see write-retry survive a chaos outage")
}
