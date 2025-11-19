package events

import (
	"context"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Core

type CoreEventHandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Queue      string
}
type CoreEventEmitOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Headers    nats.Header
}
type CoreEventsOptions struct {
	QueueGroup string

	DefaultEmitHeaders    nats.Header
	DefaultEmitMarshaller marshaller.PayloadMarshaller

	DefaultEventHandlerMarshaller marshaller.PayloadMarshaller
}
type CoreNatsEvents interface {
	NatsEvents[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, CoreEventEmitOptions]
}
type CoreEventsOptionFunc func(*CoreEventsOptions)
type CoreEventHandlerOptionFunc func(*CoreEventHandlerOptions)
type CoreEventEmitOptionFunc func(*CoreEventEmitOptions)

// JetStream

type JetStreamEventHandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
}
type JetStreamEventEmitOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Headers    nats.Header
}
type JetStreamEventsOptions struct {
	Stream       string
	DeliverGroup string

	DefaultEmitHeaders    nats.Header
	DefaultEmitMarshaller marshaller.PayloadMarshaller

	DefaultEventHandlerMarshaller marshaller.PayloadMarshaller
}
type JetStreamNatsEvents interface {
	NatsEvents[jetstream.Msg, any, JetStreamEventHandlerOptions, JetStreamEventEmitOptions]
}
type JetStreamEventsOptionFunc func(*JetStreamEventsOptions)
type JetStreamEventHandlerOptionFunc func(*JetStreamEventHandlerOptions)
type JetStreamEventEmitOptionFunc func(*JetStreamEventEmitOptions)

// Abstract interfaces

type NatsEvents[TMessage any, TAckOptFunc any, THandlerOption any, TEmitOption any] interface {
	EventRouter[TMessage, TAckOptFunc, THandlerOption, MiddlewareFunc[TMessage, TAckOptFunc]]

	StartWithContext(ctx context.Context) error
	Emit(ctx context.Context, subject string, payload any, opts ...func(*TEmitOption)) error
	Shutdown(ctx context.Context) error
}

type EventRouter[TMessage any, TAckOptFunc any, THandlerOption any, TMiddlewareFunc any] interface {
	Use(middlewares ...TMiddlewareFunc)
	AddEventHandler(subject string, handler HandlerFunc[TMessage, TAckOptFunc], opts ...func(*THandlerOption))
	Group(group string) EventRouter[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]
}

type EventContext[TMessage any, TAckOptFunc any] interface {
	context.Context
	Event(data any) error

	Headers() nats.Header
	Message() TMessage

	Ack(opts ...TAckOptFunc) error
	Nak(opts ...TAckOptFunc) error
	Term(opts ...TAckOptFunc) error
	InProgress(opts ...TAckOptFunc) error
}

type HandlerFunc[TMessage any, TAckOptFunc any] func(EventContext[TMessage, TAckOptFunc]) error
type MiddlewareFunc[TMessage any, TAckOptFunc any] func(next HandlerFunc[TMessage, TAckOptFunc]) HandlerFunc[TMessage, TAckOptFunc]
