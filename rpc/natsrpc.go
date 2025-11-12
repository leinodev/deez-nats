package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var (
	ErrInvalidSubject = errors.New("invalid subject")
	ErrRPCResponse    = errors.New("rpc responded error")
)

type natsRpcImpl struct {
	NatsRPC
	nc *nats.Conn

	rootRouter RPCRouter

	defaultHandlerOpts HandlerOptions
	defaultCallOpts    CallOptions

	mu    sync.Mutex
	subs  []*nats.Subscription
	start bool
}

// TODO: Add options builder
// TODO: Add logger
func NewNatsRPC(nc *nats.Conn, baseRoute string) NatsRPC {
	defaultHandler := HandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}
	defaultCall := CallOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}

	return &natsRpcImpl{
		nc:                 nc,
		defaultHandlerOpts: defaultHandler,
		defaultCallOpts:    defaultCall,
		rootRouter:         newRouter(baseRoute, defaultHandler),
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
func (r *natsRpcImpl) dfs() []rpcInfo {
	return r.rootRouter.dfs()
}

// Rpc methods
func (r *natsRpcImpl) StartWithContext(ctx context.Context) error {
	r.mu.Lock()
	if r.start {
		r.mu.Unlock()
		return errors.New("rpc already started")
	}
	r.start = true
	r.mu.Unlock()

	routes := r.rootRouter.dfs()
	if len(routes) == 0 {
		return nil
	}

	for _, route := range routes {
		handler := route.handler

		for _, mv := range route.middlewares {
			handler = mv(handler)
		}

		sub, err := r.nc.Subscribe(route.method, r.wrapRPCHandler(ctx, route, handler))
		if err != nil {
			r.cleanupSubscriptions()
			return fmt.Errorf("subscribe %s: %w", route.method, err)
		}

		r.mu.Lock()
		defer r.mu.Unlock()
		r.subs = append(r.subs, sub)
	}

	go func() {
		<-ctx.Done()
		r.cleanupSubscriptions()
	}()

	return nil
}

func (r *natsRpcImpl) CallRPC(ctx context.Context, subj string, request any, response any, opts CallOptions) error {
	if subj == "" {
		return fmt.Errorf("%w: empty subject", ErrInvalidSubject)
	}

	if opts.Marshaller == nil {
		opts.Marshaller = r.defaultCallOpts.Marshaller
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

func (r *natsRpcImpl) cleanupSubscriptions() {
	r.mu.Lock()
	subs := r.subs
	r.subs = nil
	r.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Drain()
	}
}
