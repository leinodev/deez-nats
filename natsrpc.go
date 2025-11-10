package natsrpcgo

import (
	"context"

	"github.com/nats-io/nats.go"
)

type RpcHandleFunc = func(c RPCContext) error
type RpcMiddlewareFunc = func(next RpcHandleFunc) RpcHandleFunc

type NatsRPC interface {
	RPCRouter

	StartWithContext(ctx context.Context) error
	CallRPC(subj string, request any, response any, optsFuncs CallOptions) error
}

type natsRpcImpl struct {
	rootRouter RPCRouter
}

func NewNatsRPC(nc *nats.Conn) NatsRPC {
	return nil
}

// Router inherited
func (r *natsRpcImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.rootRouter.Use(middlewares...)
}
func (r *natsRpcImpl) AddRPC(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc) {
	r.rootRouter.AddRPC(method, handler, opts, middlewares...)
}
func (r *natsRpcImpl) Group(group string) RPCRouter {
	return r.rootRouter.Group(group)
}
func (r *natsRpcImpl) dfs() []rpcInfo {
	return r.rootRouter.dfs()
}

// Rpc methods
func (r *natsRpcImpl) StartWithContext(ctx context.Context) error {
	return nil
}

func (r *natsRpcImpl) CallRPC(subj string, request any, response any, optsFuncs CallOptions) error {
	return nil
}
