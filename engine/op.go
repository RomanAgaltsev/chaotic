// Package engine holds the rules and decision logic that adapters consult on
// every wrapped operation. It depends only on the standard library and the
// chaotic fault package.
package engine

import "context"

// Kind identifies which adapter produced an Op.
// Do not renumber existing values.
type Kind int

// Op kind constants identify which adapter produced an Op.
const (
	OpHTTPClient Kind = iota + 1
	OpHTTPServer
	OpSQL
	OpGRPCClient
	OpGRPCServer
	OpExplicit // chaos.Point call sites
	OpPGX      // pgx adapter
	OpRedis    // go-redis adapter
	OpRabbitMQ // rabbitmq/amqp091-go adapter
	OpMongo    // mongo-driver v2 adapter
	OpKafka    // segmentio/kafka-go adapter
	OpAWS      // aws-sdk-go-v2 adapter
	OpNATS     // nats.go adapter
	OpNet      // raw net.Conn adapter
)

// Op describes a single intercepted call. Adapters construct an Op only after
// Engine.Enabled() returns true, so the no-op path allocates nothing.
type Op struct {
	Kind   Kind
	Name   string
	Method string
	Attrs  map[string]string
}

// Action is what Eval returns; adapters execute it around the wrapped call.
// Before runs prior to the call. After runs after call.
type Action interface {
	Before(ctx context.Context) error
	After(ctx context.Context) error
}

// passAction is the zero-sized, allocation-free Action returned when no rule
// matches. Pass is exported so tests can assert against it.
type passAction struct{}

func (a passAction) Before(ctx context.Context) error { return nil }
func (a passAction) After(ctx context.Context) error  { return nil }

// Pass is the canonical no-op action.
var Pass Action = passAction{}

// OutcomeReporter is an optional interface an Action may implement to receive
// the result of the wrapped call. Adapters call Outcome (when implemented)
// after the wrapped boundary returns. callErr is the wrapped call's error
// (nil or success). It is not invoked when Before short-circuits the call.
type OutcomeReporter interface {
	Outcome(ctx context.Context, callErr error)
}
