package events

import (
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

// EventsOption is a functional option for configuring EventsOptions
type EventsOption func(*EventsOptions)

func WithTimeout(timeout time.Duration) EventsOption {
	return func(opts *EventsOptions) {
		opts.DefaultHandlerOptions.JetStream.PullExpire = timeout
	}
}
func WithJetStream(enabled bool) EventsOption {
	return func(opts *EventsOptions) {
		opts.DefaultHandlerOptions.JetStream.Enabled = enabled
	}
}
func WithAutoAck(autoAck bool) EventsOption {
	return func(opts *EventsOptions) {
		opts.DefaultHandlerOptions.JetStream.AutoAck = autoAck
	}
}
func WithJetStreamContext(js nats.JetStreamContext) EventsOption {
	return func(opts *EventsOptions) {
		opts.JetStream = js
	}
}
func WithJetStreamOptions(jsOpts ...nats.JSOpt) EventsOption {
	return func(opts *EventsOptions) {
		opts.JetStreamOptions = append(opts.JetStreamOptions, jsOpts...)
	}
}
func WithDefaultMarshaller(m marshaller.PayloadMarshaller) EventsOption {
	return func(opts *EventsOptions) {
		opts.DefaultHandlerOptions.Marshaller = m
		opts.DefaultPublishOptions.Marshaller = m
	}
}
func WithDefaultQueue(queue string) EventsOption {
	return func(opts *EventsOptions) {
		opts.DefaultHandlerOptions.Queue = queue
	}
}

// EventHandlerOption is a functional option for configuring EventHandlerOptions
type EventHandlerOption func(*EventHandlerOptions)

func WithHandlerMarshaller(m marshaller.PayloadMarshaller) EventHandlerOption {
	return func(opts *EventHandlerOptions) {
		opts.Marshaller = m
	}
}
func WithHandlerQueue(queue string) EventHandlerOption {
	return func(opts *EventHandlerOptions) {
		opts.Queue = queue
	}
}
func WithHandlerJetStream(jsOpts ...JetStreamEventOption) EventHandlerOption {
	return func(opts *EventHandlerOptions) {
		for _, opt := range jsOpts {
			opt(&opts.JetStream)
		}
	}
}

// JetStreamEventOption is a functional option for configuring JetStreamEventOptions
type JetStreamEventOption func(*JetStreamEventOptions)

func WithJSEnabled(enabled bool) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.Enabled = enabled
	}
}
func WithJSAutoAck(autoAck bool) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.AutoAck = autoAck
	}
}
func WithJSPull(pull bool) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.Pull = pull
	}
}
func WithJSPullBatch(batch int) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.PullBatch = batch
	}
}
func WithJSPullExpire(expire time.Duration) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.PullExpire = expire
	}
}
func WithJSPullRetryDelay(delay time.Duration) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.PullRetryDelay = delay
	}
}
func WithJSDurable(durable string) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.Durable = durable
	}
}
func WithJSDeliverGroup(group string) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.DeliverGroup = group
	}
}
func WithJSSubjectTransform(fn func(subject string) string) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.SubjectTransform = fn
	}
}
func WithJSSubscribeOptions(subOpts ...nats.SubOpt) JetStreamEventOption {
	return func(opts *JetStreamEventOptions) {
		opts.SubscribeOptions = append(opts.SubscribeOptions, subOpts...)
	}
}

// EventPublishOption is a functional option for configuring EventPublishOptions
type EventPublishOption func(*EventPublishOptions)

func WithPublishMarshaller(m marshaller.PayloadMarshaller) EventPublishOption {
	return func(opts *EventPublishOptions) {
		opts.Marshaller = m
	}
}
func WithPublishHeader(key, value string) EventPublishOption {
	return func(opts *EventPublishOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers.Add(key, value)
	}
}
func WithPublishHeaders(headers nats.Header) EventPublishOption {
	return func(opts *EventPublishOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		for k, v := range headers {
			opts.Headers[k] = append(opts.Headers[k], v...)
		}
	}
}
func WithPublishJetStreamOptions(pubOpts ...nats.PubOpt) EventPublishOption {
	return func(opts *EventPublishOptions) {
		opts.JetStream = append(opts.JetStream, pubOpts...)
	}
}

// NewEventHandlerOptions creates EventHandlerOptions from functional options
func NewEventHandlerOptions(opts ...EventHandlerOption) EventHandlerOptions {
	handlerOpts := EventHandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
		JetStream: JetStreamEventOptions{
			AutoAck: true,
		},
	}
	for _, opt := range opts {
		opt(&handlerOpts)
	}
	return handlerOpts
}

// NewEventPublishOptions creates EventPublishOptions from functional options
func NewEventPublishOptions(opts ...EventPublishOption) EventPublishOptions {
	publishOpts := EventPublishOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
		Headers:    make(nats.Header),
		JetStream:  make([]nats.PubOpt, 0),
	}
	for _, opt := range opts {
		opt(&publishOpts)
	}
	return publishOpts
}

// NewEventsOptions creates EventsOptions from functional options
func NewEventsOptions(opts ...EventsOption) EventsOptions {
	eventsOpts := EventsOptions{
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
	}
	for _, opt := range opts {
		opt(&eventsOpts)
	}
	return eventsOpts
}
