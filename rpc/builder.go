package rpc

import (
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

// RPCOption is a functional option for configuring RPCOptions
type RPCOption func(*RPCOptions)

func WithBaseRoute(route string) RPCOption {
	return func(opts *RPCOptions) {
		opts.BaseRoute = route
	}
}
func WithQueueGroup(queueGroup string) RPCOption {
	return func(opts *RPCOptions) {
		opts.QueueGroup = queueGroup
	}
}
func WithDefaultHandlerMarshaller(m marshaller.PayloadMarshaller) RPCOption {
	return func(opts *RPCOptions) {
		opts.DefaultHandlerOptions.Marshaller = m
	}
}
func WithDefaultCallMarshaller(m marshaller.PayloadMarshaller) RPCOption {
	return func(opts *RPCOptions) {
		opts.DefaultCallOptions.Marshaller = m
	}
}
func WithDefaultCallOptions(callOpts ...CallOption) RPCOption {
	return func(opts *RPCOptions) {
		for _, opt := range callOpts {
			opt(&opts.DefaultCallOptions)
		}
	}
}
func WithDefaultHandlerOptions(handlerOpts ...HandlerOption) RPCOption {
	return func(opts *RPCOptions) {
		for _, opt := range handlerOpts {
			opt(&opts.DefaultHandlerOptions)
		}
	}
}

// HandlerOption is a functional option for configuring HandlerOptions
type HandlerOption func(*HandlerOptions)

func WithHandlerMarshaller(m marshaller.PayloadMarshaller) HandlerOption {
	return func(opts *HandlerOptions) {
		opts.Marshaller = m
	}
}

// WithHandlerMiddlewares creates a HandlerOption that sets middlewares for the handler
func WithHandlerMiddlewares(middlewares ...RpcMiddlewareFunc) HandlerOption {
	return func(opts *HandlerOptions) {
		opts.middlewares = middlewares
	}
}

// CallOption is a functional option for configuring CallOptions
type CallOption func(*CallOptions)

func WithCallMarshaller(m marshaller.PayloadMarshaller) CallOption {
	return func(opts *CallOptions) {
		opts.Marshaller = m
	}
}
func WithCallHeader(key, value string) CallOption {
	return func(opts *CallOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers.Add(key, value)
	}
}
func WithCallHeaders(headers nats.Header) CallOption {
	return func(opts *CallOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		for k, v := range headers {
			opts.Headers[k] = append(opts.Headers[k], v...)
		}
	}
}
func SetCallHeader(key string, values []string) CallOption {
	return func(opts *CallOptions) {
		if opts.Headers == nil {
			opts.Headers = make(nats.Header)
		}
		opts.Headers[key] = values
	}
}

// NewHandlerOptions creates HandlerOptions from functional options
func NewHandlerOptions(opts ...HandlerOption) HandlerOptions {
	handlerOpts := HandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}
	for _, opt := range opts {
		opt(&handlerOpts)
	}
	return handlerOpts
}

// NewCallOptions creates CallOptions from functional options
func NewCallOptions(opts ...CallOption) CallOptions {
	callOpts := CallOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
		Headers:    make(nats.Header),
	}
	for _, opt := range opts {
		opt(&callOpts)
	}
	return callOpts
}

// NewRPCOptions creates RPCOptions from functional options
func NewRPCOptions(opts ...RPCOption) RPCOptions {
	rpcOpts := RPCOptions{
		BaseRoute:  "",
		QueueGroup: "",
		DefaultHandlerOptions: HandlerOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
		},
		DefaultCallOptions: CallOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
			Headers:    make(nats.Header),
		},
	}
	for _, opt := range opts {
		opt(&rpcOpts)
	}
	return rpcOpts
}
