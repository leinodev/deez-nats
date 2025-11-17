package provider

import (
	"context"

	"github.com/nats-io/nats.go"
)

type MessagePublishOption any
type MessageSubscribeOption any

type TransportProvider interface {
	Publish(ctx context.Context, msg *nats.Msg, opts ...MessagePublishOption) error

	Subscribe(
		ctx context.Context,
		subject string,
		handler nats.MsgHandler,
		opts ...MessageSubscribeOption,
	) (TransportSubscription, error)

	QueueSubscribe(
		ctx context.Context,
		subject string,
		queue string,
		handler nats.MsgHandler,
		opts ...MessageSubscribeOption,
	) (TransportSubscription, error)
}

type TransportSubscription interface {
	Unsubscribe() error
	Drain() error
}
