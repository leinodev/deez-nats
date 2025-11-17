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
	// For core NATS, PublishMsg doesn't accept options directly
	// Options should be applied to the message before calling PublishMsg
	return p.nc.PublishMsg(msg)
}

func (p *coreProvider) Subscribe(
	ctx context.Context,
	subject string,
	handler MessageHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	sub, err := p.nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(NewTransportMessage(msg, nil))
	})
	if err != nil {
		return nil, err
	}

	return &coreSubscription{sub: sub}, nil
}

func (p *coreProvider) QueueSubscribe(
	ctx context.Context,
	subject string,
	queue string,
	handler MessageHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	sub, err := p.nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		handler(NewTransportMessage(msg, nil))
	})
	if err != nil {
		return nil, err
	}

	return &coreSubscription{sub: sub}, nil
}

type coreSubscription struct {
	sub *nats.Subscription
}

func (s *coreSubscription) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

func (s *coreSubscription) Drain() error {
	return s.sub.Drain()
}
