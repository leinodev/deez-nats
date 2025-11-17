package events

import (
	"context"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type EventHandleFunc func(EventContext) error
type EventMiddlewareFunc func(next EventHandleFunc) EventHandleFunc

type NatsEvents interface {
	EventRouter

	StartWithContext(ctx context.Context) error
	Emit(ctx context.Context, subject string, payload any, opts ...EventPublishOption) error
	Shutdown(ctx context.Context) error
}

type EventRouter interface {
	Use(middlewares ...EventMiddlewareFunc)
	AddEventHandler(subject string, handler EventHandleFunc, opts ...EventHandlerOption)
	AddEventHandlerWithMiddlewares(subject string, handler EventHandleFunc, middlewares []EventMiddlewareFunc, opts ...EventHandlerOption)
	Group(group string) EventRouter

	dfs() []eventInfo
}

type EventContext interface {
	context.Context
	Event(data any) error

	Headers() nats.Header
	Message() *nats.Msg

	Ack(opts ...nats.AckOpt) error
	Nak(opts ...nats.AckOpt) error
	Term(opts ...nats.AckOpt) error
	InProgress(opts ...nats.AckOpt) error
}

type EventsOptions struct {
	QueueGroup            string
	DefaultHandlerOptions EventHandlerOptions
	DefaultPublishOptions EventPublishOptions
	JetStream             nats.JetStreamContext
	JetStreamOptions      []nats.JSOpt
}

type EventHandlerOptions struct {
	Marshaller        marshaller.PayloadMarshaller
	Queue             string
	JetStreamInstance jetstream.JetStream
	JetStream         JetStreamEventOptions
}

type JetStreamEventOptions struct {
	Enabled          bool
	AutoAck          bool
	Pull             bool
	PullBatch        int
	PullExpire       time.Duration
	PullRetryDelay   time.Duration
	Durable          string
	DeliverGroup     string
	SubjectTransform func(subject string) string
	SubscribeOptions []nats.SubOpt
}

type EventPublishOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Headers    nats.Header
	JetStream  []nats.PubOpt
}
