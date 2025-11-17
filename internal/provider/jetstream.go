package provider

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type jetstreamProvider struct {
	js jetstream.JetStream
}

func NewJetstreamProvider(nc *nats.Conn) (TransportProvider, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream provider: %w", err)
	}
	return &jetstreamProvider{js: js}, nil
}

func (p *jetstreamProvider) Publish(
	ctx context.Context,
	msg *nats.Msg,
	opts ...MessagePublishOption,
) error {
	panic("not implemented")
}

func (p *jetstreamProvider) Subscribe(
	ctx context.Context,
	subject string,
	handler nats.MsgHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	panic("not implemented")
}

func (p *jetstreamProvider) QueueSubscribe(
	ctx context.Context,
	subject string,
	queue string,
	handler nats.MsgHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	panic("not implemented")
}
