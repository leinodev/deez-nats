package provider

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type jetstreamProvider struct {
	js     jetstream.JetStream
	stream string
}

func NewJetstreamProvider(nc *nats.Conn, stream string) (TransportProvider, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream provider: %w", err)
	}
	return &jetstreamProvider{js: js, stream: stream}, nil
}

func (p *jetstreamProvider) Publish(
	ctx context.Context,
	msg *nats.Msg,
	opts ...MessagePublishOption,
) error {
	_, err := p.js.PublishMsg(ctx, msg)
	return err
}

func (p *jetstreamProvider) Subscribe(
	ctx context.Context,
	subject string,
	handler MessageHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	// Build consumer config from options
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubjects: []string{subject},
	}

	// Use CreateOrUpdateConsumer and Consume for jetstream subscriptions
	// Note: This requires stream name, which should be provided via options or context
	// For now, we'll use a simplified approach
	jsSub, err := p.js.CreateOrUpdateConsumer(ctx, p.stream, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	consumeContext, err := jsSub.Consume(func(msg jetstream.Msg) {
		handler(NewTransportMessage(nil, msg))
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	return &jetstreamSubscription{
		consumer:   jsSub,
		consumeCtx: consumeContext,
	}, nil
}

func (p *jetstreamProvider) QueueSubscribe(
	ctx context.Context,
	subject string,
	queue string,
	handler MessageHandler,
	opts ...MessageSubscribeOption,
) (TransportSubscription, error) {
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverGroup:   queue,
	}

	jsSub, err := p.js.CreateOrUpdateConsumer(ctx, p.stream, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	consumeContext, err := jsSub.Consume(func(msg jetstream.Msg) {
		handler(NewTransportMessage(nil, msg))
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	return &jetstreamSubscription{
		consumer:   jsSub,
		consumeCtx: consumeContext,
	}, nil
}

type jetstreamSubscription struct {
	consumer   jetstream.Consumer
	consumeCtx jetstream.ConsumeContext
}

func (s *jetstreamSubscription) Unsubscribe() error {
	s.consumeCtx.Stop()
	return nil
}

func (s *jetstreamSubscription) Drain() error {
	s.consumeCtx.Drain()
	return nil
}
