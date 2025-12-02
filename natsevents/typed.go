package natsevents

import (
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TypedEventHandler is a typed event handler function
type TypedEventHandler[TMessage any, TAckOptFunc any, Payload any] func(ctx EventContext[TMessage, TAckOptFunc], payload Payload) error

// Core Events Typed Helpers

func AddTypedCoreEventHandler[Payload any](
	router EventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]],
	subject string,
	handler TypedEventHandler[*nats.Msg, nats.AckOpt, Payload],
	opts ...CoreEventHandlerOptionFunc,
) {
	wrapped := func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		var payload Payload
		if err := ctx.Event(&payload); err != nil {
			return err
		}

		return handler(ctx, payload)
	}

	// TODO: check that shit
	optsFuncs := make([]func(*CoreEventHandlerOptions), len(opts))
	for i, opt := range opts {
		optsFuncs[i] = opt
	}

	router.AddEventHandler(subject, wrapped, optsFuncs...)
}

func AddTypedCoreEventHandlerWithMarshaller[Payload any](
	router EventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]],
	subject string,
	handler TypedEventHandler[*nats.Msg, nats.AckOpt, Payload],
	marshaller marshaller.PayloadMarshaller,
	opts ...CoreEventHandlerOptionFunc,
) {
	allOpts := append([]CoreEventHandlerOptionFunc{WithCoreHandlerMarshaller(marshaller)}, opts...)
	AddTypedCoreEventHandler(router, subject, handler, allOpts...)
}

func AddTypedCoreJsonEventHandler[Payload any](
	router EventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]],
	subject string,
	handler TypedEventHandler[*nats.Msg, nats.AckOpt, Payload],
	opts ...CoreEventHandlerOptionFunc,
) {
	AddTypedCoreEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultJsonMarshaller, opts...)
}

func AddTypedCoreProtoEventHandler[Payload any](
	router EventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]],
	subject string,
	handler TypedEventHandler[*nats.Msg, nats.AckOpt, Payload],
	opts ...CoreEventHandlerOptionFunc,
) {
	AddTypedCoreEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultProtoMarshaller, opts...)
}

// JetStream Events Typed Helpers

func AddTypedJetStreamEventHandler[Payload any](
	router EventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]],
	subject string,
	handler TypedEventHandler[jetstream.Msg, any, Payload],
	opts ...JetStreamEventHandlerOptionFunc,
) {
	wrapped := func(ctx EventContext[jetstream.Msg, any]) error {
		var payload Payload
		if err := ctx.Event(&payload); err != nil {
			return err
		}

		return handler(ctx, payload)
	}

	if len(opts) > 0 {
		optsFuncs := make([]func(*JetStreamEventHandlerOptions), len(opts))
		for i, opt := range opts {
			optsFuncs[i] = opt
		}
		router.AddEventHandler(subject, wrapped, optsFuncs...)
	} else {
		router.AddEventHandler(subject, wrapped)
	}
}

func AddTypedJetStreamEventHandlerWithMarshaller[Payload any](
	router EventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]],
	subject string,
	handler TypedEventHandler[jetstream.Msg, any, Payload],
	marshaller marshaller.PayloadMarshaller,
	opts ...JetStreamEventHandlerOptionFunc,
) {
	allOpts := append([]JetStreamEventHandlerOptionFunc{WithJetStreamHandlerMarshaller(marshaller)}, opts...)
	AddTypedJetStreamEventHandler(router, subject, handler, allOpts...)
}

func AddTypedJetStreamJsonEventHandler[Payload any](
	router EventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]],
	subject string,
	handler TypedEventHandler[jetstream.Msg, any, Payload],
	opts ...JetStreamEventHandlerOptionFunc,
) {
	AddTypedJetStreamEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultJsonMarshaller, opts...)
}

func AddTypedJetStreamProtoEventHandler[Payload any](
	router EventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]],
	subject string,
	handler TypedEventHandler[jetstream.Msg, any, Payload],
	opts ...JetStreamEventHandlerOptionFunc,
) {
	AddTypedJetStreamEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultProtoMarshaller, opts...)
}
