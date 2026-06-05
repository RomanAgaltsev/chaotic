// Command rabbitmq-publish-retry demonstrates a publisher that retries through a
// transient RabbitMQ outage, and proves the retry works using the chaotic
// adapter/rabbitmq wrapper to fail publishes on demand — no real outage, no flaky
// timing.
package main

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// publisher is the minimal surface the retry helper needs; both *amqp.Channel and
// *chaosrabbitmq.Channel satisfy it.
type publisher interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// PublishWithRetry publishes msg, retrying up to attempts times with a short
// linear backoff, so a transient broker outage does not lose the message.
func PublishWithRetry(ctx context.Context, p publisher, exchange, key string, msg amqp.Publishing, attempts int) error {
	var err error
	for i := range attempts {
		if err = p.PublishWithContext(ctx, exchange, key, false, false, msg); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i+1) * 10 * time.Millisecond)
	}
	return err
}

func main() {
	fmt.Println("run `go test` in this directory to see publish-retry survive a chaos outage")
}
