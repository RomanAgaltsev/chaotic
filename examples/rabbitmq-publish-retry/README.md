# RabbitMQ publish-retry under chaos

A publisher that retries through a transient RabbitMQ outage. This example proves
the retry path actually works by using the chaotic `adapter/rabbitmq` wrapper to
make publishes fail on demand — no real outage, no flaky timing.

## What it shows

- Wrap a live connection so its channels are chaos-aware:
  `cc := chaosrabbitmq.WrapConnection(conn, eng); ch, _ := cc.Channel()`.
- A rule that `ConnDrop`s the first two publishes (`Times(2)`) simulates a
  transient broker outage. `ConnDrop` never touches the real channel, so the
  retry's next attempt lands on the still-open channel.
- `PublishWithRetry` keeps trying and the message is delivered; the rule's hit
  counter proves the outage was exercised exactly twice.

## Run it

go test ./...

The test uses [testcontainers-go](https://golang.testcontainers.org/) to start a
real RabbitMQ broker, so **Docker must be running**. Without Docker the test
skips (it does not fail).