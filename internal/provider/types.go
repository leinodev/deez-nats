package provider

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type MessagePublishOption any
type MessageSubscribeOption any

type MessageHandler func(msg TransportMessage) error

type TransportProvider interface {
	Publish(ctx context.Context, msg *nats.Msg, opts ...MessagePublishOption) error

	Subscribe(
		ctx context.Context,
		subject string,
		handler MessageHandler,
		opts ...MessageSubscribeOption,
	) (TransportSubscription, error)

	QueueSubscribe(
		ctx context.Context,
		subject string,
		queue string,
		handler MessageHandler,
		opts ...MessageSubscribeOption,
	) (TransportSubscription, error)
}

type TransportMessage interface {
	CoreMsg() *nats.Msg
	JsMsg() jetstream.Msg
}

type TransportSubscription interface {
	Unsubscribe() error
	Drain() error
}
