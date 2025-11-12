package events

import "github.com/leinodev/deez-nats/marshaller"

type TypedEventHandlerOptions struct {
	EventHandlerOptions
}

type TypedEventHandler[T any] func(ctx EventContext, payload T) error

func AddTypedEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	opts *TypedEventHandlerOptions,
	middlewares ...EventMiddlewareFunc,
) {
	if handler == nil {
		panic("typed event handler is nil")
	}

	var baseOpts *EventHandlerOptions
	if opts != nil {
		optCopy := opts.EventHandlerOptions
		baseOpts = &optCopy
	}

	wrapped := func(ctx EventContext) error {
		var payload T
		if err := ctx.Event(&payload); err != nil {
			return err
		}

		return handler(ctx, payload)
	}

	router.AddEventHandler(subject, wrapped, baseOpts, middlewares...)
}

func AddTypedEventHandlerWithMarshaller[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	marshaller marshaller.PayloadMarshaller,
	middlewares ...EventMiddlewareFunc,
) {
	opts := &TypedEventHandlerOptions{
		EventHandlerOptions: EventHandlerOptions{
			Marshaller: marshaller,
		},
	}
	AddTypedEventHandler(router, subject, handler, opts, middlewares...)
}

func AddTypedJsonEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	middlewares ...EventMiddlewareFunc,
) {
	AddTypedEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultJsonMarshaller, middlewares...)
}

func AddTypedProtoEventHandler[T any](
	router EventRouter,
	subject string,
	handler TypedEventHandler[T],
	middlewares ...EventMiddlewareFunc,
) {
	AddTypedEventHandlerWithMarshaller(router, subject, handler, marshaller.DefaultProtoMarshaller, middlewares...)
}
