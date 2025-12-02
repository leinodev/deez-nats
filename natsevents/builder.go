package natsevents

import (
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

// JetStream Events Options Builders

func WithJetStreamStream(stream string) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		opts.Stream = stream
	}
}
func WithJetStreamDeliverGroup(deliverGroup string) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		opts.DeliverGroup = deliverGroup
	}
}
func WithJetStreamDefaultEmitHeaders(headers nats.Header) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		for k, v := range headers {
			opts.DefaultEmitHeaders[k] = append(opts.DefaultEmitHeaders[k], v...)
		}
	}
}
func WithJetStreamDefaultEmitHeader(key, value string) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		opts.DefaultEmitHeaders.Add(key, value)
	}
}
func SetJetStreamDefaultEmitHeader(key string, values []string) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		opts.DefaultEmitHeaders[key] = values
	}
}
func WithJetStreamDefaultEmitMarshaller(m marshaller.PayloadMarshaller) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		opts.DefaultEmitMarshaller = m
	}
}
func WithJetStreamDefaultEventHandlerMarshaller(m marshaller.PayloadMarshaller) JetStreamEventsOptionFunc {
	return func(opts *JetStreamEventsOptions) {
		opts.DefaultEventHandlerMarshaller = m
	}
}

// JetStream Event Handler Options Builders

func WithJetStreamHandlerMarshaller(m marshaller.PayloadMarshaller) JetStreamEventHandlerOptionFunc {
	return func(opts *JetStreamEventHandlerOptions) {
		opts.Marshaller = m
	}
}

// JetStream Event Emit Options Builders

func WithJetStreamEmitMarshaller(m marshaller.PayloadMarshaller) JetStreamEventEmitOptionFunc {
	return func(opts *JetStreamEventEmitOptions) {
		opts.Marshaller = m
	}
}
func WithJetStreamEmitHeader(key, value string) JetStreamEventEmitOptionFunc {
	return func(opts *JetStreamEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers.Add(key, value)
	}
}
func WithJetStreamEmitHeaders(headers nats.Header) JetStreamEventEmitOptionFunc {
	return func(opts *JetStreamEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		for k, v := range headers {
			opts.Headers[k] = append(opts.Headers[k], v...)
		}
	}
}
func SetJetStreamEmitHeader(key string, values []string) JetStreamEventEmitOptionFunc {
	return func(opts *JetStreamEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers[key] = values
	}
}

// Core Events Options Builders

func WithCoreQueueGroup(queueGroup string) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		opts.QueueGroup = queueGroup
	}
}
func WithCoreDefaultEmitHeaders(headers nats.Header) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		for k, v := range headers {
			opts.DefaultEmitHeaders[k] = append(opts.DefaultEmitHeaders[k], v...)
		}
	}
}
func WithCoreDefaultEmitHeader(key, value string) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		opts.DefaultEmitHeaders.Add(key, value)
	}
}
func SetCoreDefaultEmitHeader(key string, values []string) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		if opts.DefaultEmitHeaders == nil {
			opts.DefaultEmitHeaders = make(nats.Header)
		}
		opts.DefaultEmitHeaders[key] = values
	}
}
func WithCoreDefaultEmitMarshaller(m marshaller.PayloadMarshaller) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		opts.DefaultEmitMarshaller = m
	}
}
func WithCoreDefaultEventHandlerMarshaller(m marshaller.PayloadMarshaller) CoreEventsOptionFunc {
	return func(opts *CoreEventsOptions) {
		opts.DefaultEventHandlerMarshaller = m
	}
}

// Core Event Handler Options Builders

func WithCoreHandlerMarshaller(m marshaller.PayloadMarshaller) CoreEventHandlerOptionFunc {
	return func(opts *CoreEventHandlerOptions) {
		opts.Marshaller = m
	}
}
func WithCoreHandlerQueue(queue string) CoreEventHandlerOptionFunc {
	return func(opts *CoreEventHandlerOptions) {
		opts.Queue = queue
	}
}

// Core Event Emit Options Builders

func WithCoreEmitMarshaller(m marshaller.PayloadMarshaller) CoreEventEmitOptionFunc {
	return func(opts *CoreEventEmitOptions) {
		opts.Marshaller = m
	}
}
func WithCoreEmitHeader(key, value string) CoreEventEmitOptionFunc {
	return func(opts *CoreEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers.Add(key, value)
	}
}
func WithCoreEmitHeaders(headers nats.Header) CoreEventEmitOptionFunc {
	return func(opts *CoreEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		for k, v := range headers {
			opts.Headers[k] = append(opts.Headers[k], v...)
		}
	}
}
func SetCoreEmitHeader(key string, values []string) CoreEventEmitOptionFunc {
	return func(opts *CoreEventEmitOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers[key] = values
	}
}
