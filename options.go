package natsrpc

import (
	"time"

	"github.com/nats-io/nats.go"
)

type NatsRPCOptions struct {
	BaseRoute             string
	WorkersCount          int
	MsgPoolSize           int
	DefaultHandlerOptions HandlerOptions
	DefaultCallOptions    CallOptions
}

func NewRPCOptions(opts ...NatsRPCOption) NatsRPCOptions {
	rpcOpts := NatsRPCOptions{
		BaseRoute:             "",
		WorkersCount:          100,
		MsgPoolSize:           1000,
		DefaultHandlerOptions: NewHandlerOptions(),
		DefaultCallOptions:    NewCallOptions(),
	}
	for _, opt := range opts {
		opt(&rpcOpts)
	}
	return rpcOpts
}

type NatsRPCOption func(*NatsRPCOptions)

func WithBaseRoute(route string) NatsRPCOption {
	return func(opts *NatsRPCOptions) {
		opts.BaseRoute = route
	}
}
func WithWorkersCount(workers int) NatsRPCOption {
	return func(opts *NatsRPCOptions) {
		opts.WorkersCount = workers
	}
}
func WithMsgPoolSize(msgPoolSize int) NatsRPCOption {
	return func(opts *NatsRPCOptions) {
		opts.MsgPoolSize = msgPoolSize
	}
}

// HandlerOption is a functional option for configuring HandlerOptions
type HandlerOptions struct {
	Timeout time.Duration
}

func NewHandlerOptions(opts ...HandlerOption) HandlerOptions {
	handlerOpts := HandlerOptions{
		Timeout: time.Minute,
	}

	for _, opt := range opts {
		opt(&handlerOpts)
	}
	return handlerOpts
}

type HandlerOption func(*HandlerOptions)

func WithHandlerTimeout(timeout time.Duration) HandlerOption {
	return func(opts *HandlerOptions) {
		opts.Timeout = timeout
	}
}

// CallOption is a functional option for configuring CallOptions
type CallOptions struct {
	Encoding string
	Headers  nats.Header
}

func NewCallOptions(opts ...CallOption) CallOptions {
	callOpts := CallOptions{
		Encoding: "json",
		Headers:  make(nats.Header),
	}

	for _, opt := range opts {
		opt(&callOpts)
	}
	return callOpts
}

type CallOption func(*CallOptions)

func WithEncoding(e string) CallOption {
	return func(opts *CallOptions) {
		opts.Encoding = e
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
