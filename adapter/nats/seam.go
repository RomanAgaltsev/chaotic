package nats

import (
	"time"

	natsgo "github.com/nats-io/nats.go"
)

// natsConn is the faultable subset of *nats.Conn that the chaos wrapper
// intercepts. *nats.Conn satisfies it directly; unit tests supply a fake. Every
// other *nats.Conn method (Flush, Status, Close, LastError, ...) reaches callers
// via the embedded *nats.Conn in Conn.
type natsConn interface {
	Publish(subj string, data []byte) error
	Request(subj string, data []byte, timeout time.Duration) (*natsgo.Msg, error)
	Subscribe(subj string, cb natsgo.MsgHandler) (*natsgo.Subscription, error)
	QueueSubscribe(subj, queue string, cb natsgo.MsgHandler) (*natsgo.Subscription, error)
	Drain() error
}
