package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

const defaultRequestTimeout = 5 * time.Second

type natsRpcImpl struct {
	nc *nats.Conn

	rootRouter RPCRouter

	defaultHandlerOpts HandlerOptions
	defaultCallOpts    CallOptions

	mu    sync.Mutex
	subs  []*nats.Subscription
	start bool
}

func NewNatsRPC(nc *nats.Conn) NatsRPC {
	defaultMarshaller := marshaller.DefaultJsonMarshaller

	defaultHandler := HandlerOptions{
		Marshaller: defaultMarshaller,
	}
	defaultCall := CallOptions{
		Marshaller: defaultMarshaller,
	}

	return &natsRpcImpl{
		nc:                 nc,
		defaultHandlerOpts: defaultHandler,
		defaultCallOpts:    defaultCall,
		rootRouter:         newRouter("", defaultHandler),
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
		if route.options.Marshaller == nil {
			route.options.Marshaller = r.defaultHandlerOpts.Marshaller
		}

		handler := route.handler
		for i := len(route.middlewares) - 1; i >= 0; i-- {
			handler = route.middlewares[i](handler)
		}

		sub, err := r.nc.Subscribe(route.method, r.wrapRPCHandler(ctx, route, handler))
		if err != nil {
			r.cleanupSubscriptions()
			return fmt.Errorf("subscribe %s: %w", route.method, err)
		}

		r.trackSubscription(sub)
	}

	go func() {
		<-ctx.Done()
		r.cleanupSubscriptions()
	}()

	return nil
}

func (r *natsRpcImpl) CallRPC(subj string, request any, response any, opts CallOptions) error {
	if subj == "" {
		return errors.New("empty subject")
	}

	options := r.defaultCallOpts
	if opts.Marshaller != nil {
		options.Marshaller = opts.Marshaller
	}
	if options.Marshaller == nil {
		options.Marshaller = marshaller.DefaultJsonMarshaller
	}

	payload, err := options.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: request,
	})
	if err != nil {
		return fmt.Errorf("marshall request: %w", err)
	}

	timeout := r.nc.Opts.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	msg, err := r.nc.Request(subj, payload, timeout)
	if err != nil {
		return err
	}

	respObj := &marshaller.MarshalObject{
		Data: response,
	}

	if err := options.Marshaller.Unmarshall(msg.Data, respObj); err != nil {
		return fmt.Errorf("unmarshall response: %w", err)
	}

	if respObj.Error != "" {
		return errors.New(respObj.Error)
	}

	return nil
}

func (r *natsRpcImpl) wrapRPCHandler(ctx context.Context, info rpcInfo, handler RpcHandleFunc) nats.MsgHandler {
	return func(msg *nats.Msg) {
		if msg == nil {
			return
		}

		writer := func(obj *marshaller.MarshalObject, headers nats.Header) error {
			if msg.Reply == "" {
				return errors.New("rpc call without reply subject")
			}

			data, err := info.options.Marshaller.Marshall(obj)
			if err != nil {
				return err
			}

			resp := nats.NewMsg(msg.Reply)
			resp.Data = data
			if len(headers) > 0 {
				resp.Header = cloneHeader(headers)
			}

			return msg.RespondMsg(resp)
		}

		rpcCtx := newRpcContext(ctx, msg, info.options.Marshaller, writer)

		ctxImpl, _ := rpcCtx.(*rpcContextImpl)

		if err := handler(rpcCtx); err != nil {
			if ctxImpl != nil && !ctxImpl.hasResponse() {
				if werr := ctxImpl.writeError(err); werr != nil && !errors.Is(werr, errResponseAlreadyWritten) {
					// ignore marshalling error, will trigger NAK
					_ = werr
				}
			}

			if ctxImpl != nil && ctxImpl.hasResponse() {
				ackMsg(msg)
				return
			}

			nakMsg(msg)
			return
		}

		if ctxImpl != nil && ctxImpl.hasResponse() {
			ackMsg(msg)
			return
		}

		if err := rpcCtx.Ok(nil); err != nil {
			if errors.Is(err, errResponseAlreadyWritten) {
				ackMsg(msg)
				return
			}

			if ctxImpl != nil {
				_ = ctxImpl.writeError(err)
			}

			nakMsg(msg)
			return
		}

		ackMsg(msg)
	}
}

func (r *natsRpcImpl) trackSubscription(sub *nats.Subscription) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.subs = append(r.subs, sub)
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

func ackMsg(msg *nats.Msg) {
	if msg == nil {
		return
	}
	_ = msg.Ack()
}

func nakMsg(msg *nats.Msg) {
	if msg == nil {
		return
	}
	_ = msg.Nak()
}
