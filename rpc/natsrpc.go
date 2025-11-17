package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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

	subsTracker *subscriptions.Tracker

	handlersWatch sync.WaitGroup
}

// TODO: Add logger
func NewNatsRPC(nc *nats.Conn, opts ...RPCOption) NatsRPC {
	options := NewRPCOptions(opts...)

	return &natsRpcImpl{
		nc:          nc,
		options:     options,
		rootRouter:  newRouter(options.BaseRoute, options.DefaultHandlerOptions),
		subsTracker: subscriptions.NewTracker(),
	}
}

// Router inherited
func (r *natsRpcImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.rootRouter.Use(middlewares...)
}
func (r *natsRpcImpl) AddRPCHandler(method string, handler RpcHandleFunc, opts ...HandlerOption) {
	r.rootRouter.AddRPCHandler(method, handler, opts...)
}
func (r *natsRpcImpl) AddRPCHandlerWithMiddlewares(method string, handler RpcHandleFunc, middlewares []RpcMiddlewareFunc, opts ...HandlerOption) {
	r.rootRouter.AddRPCHandlerWithMiddlewares(method, handler, middlewares, opts...)
}
func (r *natsRpcImpl) Group(group string) RPCRouter {
	return r.rootRouter.Group(group)
}

// Rpc methods
func (r *natsRpcImpl) StartWithContext(ctx context.Context) error {
	handler := func(route rpcInfo) nats.MsgHandler {
		return r.wrapRPCHandler(
			ctx,
			route,
			middleware.Apply(route.handler, route.middlewares, true),
		)
	}

	for _, route := range r.rootRouter.dfs() {
		var sub *nats.Subscription
		var err error

		if r.options.QueueGroup != "" {
			sub, err = r.nc.QueueSubscribe(
				route.method,
				r.options.QueueGroup,
				handler(route),
			)
		} else {
			sub, err = r.nc.Subscribe(
				route.method,
				handler(route),
			)
		}

		if err != nil {
			r.Shutdown(ctx)

			return fmt.Errorf("failed to subscribe %s: %w", route.method, err)
		}

		r.subsTracker.Track(sub)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = r.Shutdown(shutdownCtx)
	}()

	return nil
}

func (r *natsRpcImpl) Shutdown(ctx context.Context) error {
	r.subsTracker.Drain() // Drain subscriptions to stop accepting new messages

	finished := make(chan struct{})
	go func() {
		r.handlersWatch.Wait() // Wait for all active handlers to finish.
		<-finished
		close(finished)
	}()

	select {
	case <-finished:
		break
	case <-ctx.Done():
		return fmt.Errorf("failed to wait for handlers finish: %w", context.DeadlineExceeded)
	}

	r.subsTracker.Unsubscribe() // Unsubscribe from all routes
	return nil
}

func (r *natsRpcImpl) CallRPC(ctx context.Context, subj string, request any, response any, opts ...CallOption) error {
	if subj == "" {
		return fmt.Errorf("%w: empty subject", ErrInvalidSubject)
	}

	callOpts := r.options.DefaultCallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	if callOpts.Marshaller == nil {
		callOpts.Marshaller = r.options.DefaultCallOptions.Marshaller
	}

	payload, err := callOpts.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: request,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := r.nc.RequestMsgWithContext(ctx, &nats.Msg{
		Subject: subj,
		Data:    payload,
		Header:  callOpts.Headers,
	})
	if err != nil {
		return err
	}

	respObj := &marshaller.MarshalObject{
		Data: response,
	}

	if err := callOpts.Marshaller.Unmarshall(msg.Data, respObj); err != nil {
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
		r.handlersWatch.Add(1)
		defer r.handlersWatch.Done()

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
