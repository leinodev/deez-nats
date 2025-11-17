package provider

import (
	"context"

	"github.com/nats-io/nats.go"
)

type coreProvider struct {
	nc *nats.Conn
}

func NewCoreProvider(nc *nats.Conn) TransportProvider {
	return &coreProvider{nc: nc}
}

func (p *coreProvider) Publish(ctx context.Context, msg *nats.Msg, opts ...MessagePublishOption) error {
	panic("not implemented")
}

func (p *coreProvider) Subscribe(
	ctx context.Context,
	subject string,
	handler nats.MsgHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	panic("not implemented")
}

func (p *coreProvider) QueueSubscribe(
	ctx context.Context,
	subject string,
	queue string,
	handler nats.MsgHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	panic("not implemented")
}
