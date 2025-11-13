package rpc

import (
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type HandlerOptionsBuilder struct {
	opts HandlerOptions
}

// NewHandlerOptionsBuilder creates a new HandlerOptionsBuilder
func NewHandlerOptionsBuilder() *HandlerOptionsBuilder {
	return &HandlerOptionsBuilder{
		opts: HandlerOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
		},
	}
}
func (b *HandlerOptionsBuilder) WithMarshaller(m marshaller.PayloadMarshaller) *HandlerOptionsBuilder {
	b.opts.Marshaller = m
	return b
}
func (b *HandlerOptionsBuilder) Build() HandlerOptions {
	return b.opts
}

type CallOptionsBuilder struct {
	opts CallOptions
}

// NewCallOptionsBuilder creates a new CallOptionsBuilder
func NewCallOptionsBuilder() *CallOptionsBuilder {
	return &CallOptionsBuilder{
		opts: CallOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
			Headers:    make(nats.Header),
		},
	}
}
func (b *CallOptionsBuilder) WithMarshaller(m marshaller.PayloadMarshaller) *CallOptionsBuilder {
	b.opts.Marshaller = m
	return b
}
func (b *CallOptionsBuilder) WithHeader(key, value string) *CallOptionsBuilder {
	if b.opts.Headers == nil {
		b.opts.Headers = make(nats.Header)
	}
	b.opts.Headers.Add(key, value)
	return b
}
func (b *CallOptionsBuilder) WithHeaders(headers nats.Header) *CallOptionsBuilder {
	if b.opts.Headers == nil {
		b.opts.Headers = make(nats.Header)
	}
	for k, v := range headers {
		b.opts.Headers[k] = append(b.opts.Headers[k], v...)
	}
	return b
}
func (b *CallOptionsBuilder) SetHeader(key string, values []string) *CallOptionsBuilder {
	if b.opts.Headers == nil {
		b.opts.Headers = make(nats.Header)
	}
	b.opts.Headers[key] = values
	return b
}
func (b *CallOptionsBuilder) Build() CallOptions {
	return b.opts
}

type RPCOptionsBuilder struct {
	opts RPCOptions
}

// NewRPCOptionsBuilder creates a new RPCOptionsBuilder
func NewRPCOptionsBuilder() *RPCOptionsBuilder {
	return &RPCOptionsBuilder{
		opts: RPCOptions{
			BaseRoute: "",
			DefaultHandlerOptions: HandlerOptions{
				Marshaller: marshaller.DefaultJsonMarshaller,
			},
			DefaultCallOptions: CallOptions{
				Marshaller: marshaller.DefaultJsonMarshaller,
				Headers:    make(nats.Header),
			},
		},
	}
}
func (b *RPCOptionsBuilder) WithBaseRoute(route string) *RPCOptionsBuilder {
	b.opts.BaseRoute = route
	return b
}
func (b *RPCOptionsBuilder) WithDefaultHandlerOptions(fn func(*HandlerOptionsBuilder)) *RPCOptionsBuilder {
	handlerBuilder := NewHandlerOptionsBuilder()
	handlerBuilder.opts = b.opts.DefaultHandlerOptions
	fn(handlerBuilder)
	b.opts.DefaultHandlerOptions = handlerBuilder.Build()
	return b
}
func (b *RPCOptionsBuilder) WithDefaultCallOptions(fn func(*CallOptionsBuilder)) *RPCOptionsBuilder {
	callBuilder := NewCallOptionsBuilder()
	callBuilder.opts = b.opts.DefaultCallOptions
	fn(callBuilder)
	b.opts.DefaultCallOptions = callBuilder.Build()
	return b
}
func (b *RPCOptionsBuilder) Build() RPCOptions {
	return b.opts
}
