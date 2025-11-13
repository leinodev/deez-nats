package events

import "github.com/leinodev/deez-nats/marshaller"

type TypedEventHandler[T any] func(ctx EventContext, payload T) error

func AddTypedEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	opts ...EventHandlerOption,
) {
	if handler == nil {
		panic("typed event handler is nil")
	}

	wrapped := func(ctx EventContext) error {
		var payload T
		if err := ctx.Event(&payload); err != nil {
			return err
		}

		return handler(ctx, payload)
	}

	router.AddEventHandler(subject, wrapped, opts...)
}

func AddTypedEventHandlerWithMarshaller[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	marshaller marshaller.PayloadMarshaller,
	opts ...EventHandlerOption,
) {
	allOpts := append([]EventHandlerOption{WithHandlerMarshaller(marshaller)}, opts...)
	AddTypedEventHandler(router, subject, handler, allOpts...)
}

func AddTypedJsonEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	opts ...EventHandlerOption,
) {
	AddTypedEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultJsonMarshaller, opts...)
}

func AddTypedProtoEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	opts ...EventHandlerOption,
) {
	AddTypedEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultProtoMarshaller, opts...)
}
