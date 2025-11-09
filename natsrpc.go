package natsrpcgo

import "context"

type RpcHandleFunc = func(c RPCContext) error
type RpcMiddlewareFunc = func(next RpcHandleFunc) RpcHandleFunc

type HandlerOptions = struct{}

type CallOptions = struct{}

type NatsRPC interface {
	RPCRouter

	StartWithContext(ctx context.Context) error
	CallRPC(subj string, request any, response any, optsFuncs CallOptions) error
}
