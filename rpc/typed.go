package rpc

import "github.com/leinodev/deez-nats/marshaller"

type TypedRPCHandlerOptions struct {
	HandlerOptions
}

type TypedRPCHandler[Req any, Resp any] func(ctx RPCContext, request Req) (Resp, error)

func AddTypedRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	opts *TypedRPCHandlerOptions,
	middlewares ...RpcMiddlewareFunc,
) {
	if handler == nil {
		panic("typed rpc handler is nil")
	}

	var baseOpts *HandlerOptions
	if opts != nil {
		optCopy := opts.HandlerOptions
		baseOpts = &optCopy
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

	router.AddRPCHandler(method, wrapped, baseOpts, middlewares...)
}

func AddTypedRPCHandlerWithMarshaller[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	marshaller marshaller.PayloadMarshaller,
	middlewares ...RpcMiddlewareFunc,
) {
	opts := &TypedRPCHandlerOptions{
		HandlerOptions: HandlerOptions{
			Marshaller: marshaller,
		},
	}
	AddTypedRPCHandler(router, method, handler, opts, middlewares...)
}

func AddTypedJsonRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	middlewares ...RpcMiddlewareFunc,
) {
	AddTypedRPCHandlerWithMarshaller(router, method, handler, marshaller.DefaultJsonMarshaller, middlewares...)
}

func AddTypedProtoRPCHandler[Req any, Resp any](
	router RPCRouter,
	method string,
	handler TypedRPCHandler[Req, Resp],
	middlewares ...RpcMiddlewareFunc,
) {
	AddTypedRPCHandlerWithMarshaller(router, method, handler, marshaller.DefaultProtoMarshaller, middlewares...)
}
