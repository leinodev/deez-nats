package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/leinodev/deez-nats/internal/lifecycle"
	"github.com/leinodev/deez-nats/internal/middleware"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var (
	ErrInvalidSubject = errors.New("invalid subject")
	ErrRPCResponse    = errors.New("rpc responded error")
)

type natsRpcImpl struct {
	NatsRPC
	nc      *nats.Conn
	options RPCOptions

	rootRouter RPCRouter

	lifecycleMgr    *lifecycle.Manager
	subscriptionMgr *subscriptions.Manager
}

// TODO: Add logger
func NewNatsRPC(nc *nats.Conn, opts *RPCOptions) NatsRPC {
	if opts == nil {
		defaultOpts := NewRPCOptionsBuilder().Build()
		opts = &defaultOpts
	}

	return &natsRpcImpl{
		nc:              nc,
		options:         *opts,
		rootRouter:      newRouter(opts.BaseRoute, opts.DefaultHandlerOptions),
		lifecycleMgr:    lifecycle.NewManager(),
		subscriptionMgr: subscriptions.NewManager(),
	}
}

// Router inherited
func (r *natsRpcImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.rootRouter.Use(middlewares...)
}
func (r *natsRpcImpl) AddRPCHandler(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc) {
	r.rootRouter.AddRPCHandler(method, handler, opts, middlewares...)
}
func (r *natsRpcImpl) Group(group string) RPCRouter {
	return r.rootRouter.Group(group)
}

// Rpc methods
func (r *natsRpcImpl) StartWithContext(ctx context.Context) error {
	if err := r.lifecycleMgr.MarkAsStarted(); err != nil {
		return fmt.Errorf("rpc: %w", err)
	}

	if err := r.bindAllRoutes(ctx); err != nil {
		r.subscriptionMgr.Cleanup()
		return err
	}

	go func() {
		<-ctx.Done()
		r.subscriptionMgr.Cleanup()
	}()

	return nil
}

func (r *natsRpcImpl) bindAllRoutes(ctx context.Context) error {
	routes := r.rootRouter.dfs()
	if len(routes) == 0 {
		return nil
	}

	for _, route := range routes {
		if err := r.bindRoute(ctx, route); err != nil {
			return err
		}
	}

	return nil
}

func (r *natsRpcImpl) bindRoute(ctx context.Context, route rpcInfo) error {
	handler := middleware.Apply(route.handler, route.middlewares, true)
	msgHandler := r.wrapRPCHandler(ctx, route, handler)

	sub, err := r.nc.Subscribe(route.method, msgHandler)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", route.method, err)
	}

	r.subscriptionMgr.Track(sub)
	return nil
}

func (r *natsRpcImpl) CallRPC(ctx context.Context, subj string, request any, response any, opts CallOptions) error {
	if subj == "" {
		return fmt.Errorf("%w: empty subject", ErrInvalidSubject)
	}

	if opts.Marshaller == nil {
		opts.Marshaller = r.options.DefaultCallOptions.Marshaller
	}

	payload, err := opts.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: request,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := r.nc.RequestMsgWithContext(ctx, &nats.Msg{
		Subject: subj,
		Data:    payload,
		Header:  opts.Headers,
	})
	if err != nil {
		return err
	}

	respObj := &marshaller.MarshalObject{
		Data: response,
	}

	if err := opts.Marshaller.Unmarshall(msg.Data, respObj); err != nil {
		return fmt.Errorf("unmarshall response: %w", err)
	}

	if respObj.Error != "" {
		return fmt.Errorf("%w: %s", ErrRPCResponse, respObj.Error)
	}
	return nil
}

func (r *natsRpcImpl) wrapRPCHandler(ctx context.Context, info rpcInfo, handler RpcHandleFunc) nats.MsgHandler {
	return func(msg *nats.Msg) {
		if msg == nil {
			return
		}
		rpcCtx := newRpcContext(ctx, msg, info.options)

		err := handler(rpcCtx)
		if rpcCtx.responseWritten() {
			_ = msg.Ack()
			return
		}

		if err == nil {
			// if no error, repond with empty data
			err = rpcCtx.Ok(nil)
		}
		if err != nil {
			// Got an error, respond with error
			_ = rpcCtx.writeError(err)
		}
	}
}
