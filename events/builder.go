package events

import (
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type EventHandlerOptionsBuilder struct {
	opts EventHandlerOptions
}

// NewEventHandlerOptionsBuilder creates a new EventHandlerOptionsBuilder
func NewEventHandlerOptionsBuilder() *EventHandlerOptionsBuilder {
	return &EventHandlerOptionsBuilder{
		opts: EventHandlerOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
			JetStream: JetStreamEventOptions{
				AutoAck: true,
			},
		},
	}
}
func (b *EventHandlerOptionsBuilder) WithMarshaller(m marshaller.PayloadMarshaller) *EventHandlerOptionsBuilder {
	b.opts.Marshaller = m
	return b
}
func (b *EventHandlerOptionsBuilder) WithQueue(queue string) *EventHandlerOptionsBuilder {
	b.opts.Queue = queue
	return b
}
func (b *EventHandlerOptionsBuilder) WithJetStream(fn func(*JetStreamEventOptionsBuilder)) *EventHandlerOptionsBuilder {
	jsBuilder := NewJetStreamEventOptionsBuilder()
	jsBuilder.opts = b.opts.JetStream
	fn(jsBuilder)
	b.opts.JetStream = jsBuilder.Build()
	return b
}
func (b *EventHandlerOptionsBuilder) Build() EventHandlerOptions {
	return b.opts
}

type JetStreamEventOptionsBuilder struct {
	opts JetStreamEventOptions
}

const (
	defaultPullBatchSize    = 1
	defaultPullFetchMaxWait = time.Second
	defaultPullRetryDelay   = 100 * time.Millisecond
)

// NewJetStreamEventOptionsBuilder creates a new JetStreamEventOptionsBuilder
func NewJetStreamEventOptionsBuilder() *JetStreamEventOptionsBuilder {
	return &JetStreamEventOptionsBuilder{
		opts: JetStreamEventOptions{
			AutoAck:          true,
			PullBatch:        defaultPullBatchSize,
			PullExpire:       defaultPullFetchMaxWait,
			PullRetryDelay:   defaultPullRetryDelay,
			SubscribeOptions: make([]nats.SubOpt, 0),
		},
	}
}
func (b *JetStreamEventOptionsBuilder) Enabled() *JetStreamEventOptionsBuilder {
	b.opts.Enabled = true
	return b
}
func (b *JetStreamEventOptionsBuilder) Disabled() *JetStreamEventOptionsBuilder {
	b.opts.Enabled = false
	return b
}
func (b *JetStreamEventOptionsBuilder) WithAutoAck(autoAck bool) *JetStreamEventOptionsBuilder {
	b.opts.AutoAck = autoAck
	return b
}
func (b *JetStreamEventOptionsBuilder) WithPull(pull bool) *JetStreamEventOptionsBuilder {
	b.opts.Pull = pull
	return b
}
func (b *JetStreamEventOptionsBuilder) WithPullBatch(batch int) *JetStreamEventOptionsBuilder {
	b.opts.PullBatch = batch
	return b
}
func (b *JetStreamEventOptionsBuilder) WithPullExpire(expire time.Duration) *JetStreamEventOptionsBuilder {
	b.opts.PullExpire = expire
	return b
}
func (b *JetStreamEventOptionsBuilder) WithPullRetryDelay(delay time.Duration) *JetStreamEventOptionsBuilder {
	b.opts.PullRetryDelay = delay
	return b
}
func (b *JetStreamEventOptionsBuilder) WithDurable(durable string) *JetStreamEventOptionsBuilder {
	b.opts.Durable = durable
	return b
}
func (b *JetStreamEventOptionsBuilder) WithDeliverGroup(group string) *JetStreamEventOptionsBuilder {
	b.opts.DeliverGroup = group
	return b
}
func (b *JetStreamEventOptionsBuilder) WithSubjectTransform(fn func(subject string) string) *JetStreamEventOptionsBuilder {
	b.opts.SubjectTransform = fn
	return b
}
func (b *JetStreamEventOptionsBuilder) WithSubscribeOptions(opts ...nats.SubOpt) *JetStreamEventOptionsBuilder {
	b.opts.SubscribeOptions = append(b.opts.SubscribeOptions, opts...)
	return b
}
func (b *JetStreamEventOptionsBuilder) Build() JetStreamEventOptions {
	return b.opts
}

type EventPublishOptionsBuilder struct {
	opts EventPublishOptions
}

// NewEventPublishOptionsBuilder creates a new EventPublishOptionsBuilder
func NewEventPublishOptionsBuilder() *EventPublishOptionsBuilder {
	return &EventPublishOptionsBuilder{
		opts: EventPublishOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
			Headers:    make(nats.Header),
			JetStream:  make([]nats.PubOpt, 0),
		},
	}
}
func (b *EventPublishOptionsBuilder) WithMarshaller(m marshaller.PayloadMarshaller) *EventPublishOptionsBuilder {
	b.opts.Marshaller = m
	return b
}
func (b *EventPublishOptionsBuilder) WithHeader(key, value string) *EventPublishOptionsBuilder {
	if b.opts.Headers == nil {
		b.opts.Headers = make(nats.Header)
	}
	b.opts.Headers.Add(key, value)
	return b
}
func (b *EventPublishOptionsBuilder) WithHeaders(headers nats.Header) *EventPublishOptionsBuilder {
	if b.opts.Headers == nil {
		b.opts.Headers = make(nats.Header)
	}
	for k, v := range headers {
		b.opts.Headers[k] = append(b.opts.Headers[k], v...)
	}
	return b
}
func (b *EventPublishOptionsBuilder) WithJetStreamOptions(opts ...nats.PubOpt) *EventPublishOptionsBuilder {
	b.opts.JetStream = append(b.opts.JetStream, opts...)
	return b
}
func (b *EventPublishOptionsBuilder) Build() EventPublishOptions {
	return b.opts
}

type EventsOptionsBuilder struct {
	opts EventsOptions
}

// NewEventsOptionsBuilder creates a new EventsOptionsBuilder
func NewEventsOptionsBuilder() *EventsOptionsBuilder {
	return &EventsOptionsBuilder{
		opts: EventsOptions{
			DefaultHandlerOptions: EventHandlerOptions{
				Marshaller: marshaller.DefaultJsonMarshaller,
				JetStream: JetStreamEventOptions{
					AutoAck: true,
				},
			},
			DefaultPublishOptions: EventPublishOptions{
				Marshaller: marshaller.DefaultJsonMarshaller,
			},
			JetStreamOptions: make([]nats.JSOpt, 0),
		},
	}
}
func (b *EventsOptionsBuilder) WithDefaultHandlerOptions(fn func(*EventHandlerOptionsBuilder)) *EventsOptionsBuilder {
	handlerBuilder := NewEventHandlerOptionsBuilder()
	handlerBuilder.opts = b.opts.DefaultHandlerOptions
	fn(handlerBuilder)
	b.opts.DefaultHandlerOptions = handlerBuilder.Build()
	return b
}
func (b *EventsOptionsBuilder) WithDefaultPublishOptions(fn func(*EventPublishOptionsBuilder)) *EventsOptionsBuilder {
	publishBuilder := NewEventPublishOptionsBuilder()
	publishBuilder.opts = b.opts.DefaultPublishOptions
	fn(publishBuilder)
	b.opts.DefaultPublishOptions = publishBuilder.Build()
	return b
}
func (b *EventsOptionsBuilder) WithJetStream(js nats.JetStreamContext) *EventsOptionsBuilder {
	b.opts.JetStream = js
	return b
}
func (b *EventsOptionsBuilder) WithJetStreamOptions(opts ...nats.JSOpt) *EventsOptionsBuilder {
	b.opts.JetStreamOptions = append(b.opts.JetStreamOptions, opts...)
	return b
}
func (b *EventsOptionsBuilder) Build() EventsOptions {
	return b.opts
}
