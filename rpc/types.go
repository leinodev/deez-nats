package rpc

import (
	"context"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type RpcHandleFunc func(c RPCContext) error
type RpcMiddlewareFunc func(next RpcHandleFunc) RpcHandleFunc

type NatsRPC interface {
	RPCRouter

	StartWithContext(ctx context.Context) error
	CallRPC(ctx context.Context, subj string, request any, response any, opts CallOptions) error
}

type RPCRouter interface {
	Use(middlewares ...RpcMiddlewareFunc)
	AddRPCHandler(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc)
	Group(group string) RPCRouter

	dfs() []rpcInfo
}

type RPCContext interface {
	context.Context
	Request(data any) error

	Ok(data any) error

	RequestHeaders() nats.Header
	Headers() nats.Header

	responseWritten() bool

	writeError(err error) error
}

type HandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
}

type CallOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Headers    nats.Header
}

type RPCOptions struct {
	BaseRoute             string
	DefaultHandlerOptions HandlerOptions
	DefaultCallOptions    CallOptions
}
