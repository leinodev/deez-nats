package natsrpc

import "github.com/leinodev/deez-nats/marshaller"

type TypedRPCHandler[Req any, Resp any] func(ctx RPCContext, request Req) (Resp, error)

func AddTypedRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	opts ...HandlerOption,
) {
	if handler == nil {
		panic("typed rpc handler is nil")
	}

	wrapped := func(ctx RPCContext) error {
		var request Req
		if err := ctx.Request(&request); err != nil {
			return err
		}

		response, err := handler(ctx, request)
		if err != nil {
			return err
		}

		return ctx.Ok(response)
	}

	if len(opts) > 0 {
		router.AddRPCHandler(method, wrapped, opts...)
	} else {
		router.AddRPCHandler(method, wrapped)
	}
}

func AddTypedRPCHandlerWithMarshaller[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	marshaller marshaller.PayloadMarshaller,
	opts ...HandlerOption,
) {
	allOpts := append([]HandlerOption{WithHandlerMarshaller(marshaller)}, opts...)
	AddTypedRPCHandler(router, method, handler, allOpts...)
}

func AddTypedJsonRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	opts ...HandlerOption,
) {
	AddTypedRPCHandlerWithMarshaller(router, method, handler, marshaller.DefaultJsonMarshaller, opts...)
}

func AddTypedProtoRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	opts ...HandlerOption,
) {
	AddTypedRPCHandlerWithMarshaller(router, method, handler, marshaller.DefaultProtoMarshaller, opts...)
}
