package natsrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Routing

type RPCHandleFunc func(c NatsRPCContext) error

type Route struct {
	Path        string
	Handler     RPCHandleFunc
	Opts        RPCHandlerOptions
	Subscripion *nats.Subscription
}

// NatsRPC

type NatsRPC interface {
	AddRPC(method string, handler RPCHandleFunc, optsFuncs ...HandlerOptionFunc)
	StartWithContext(ctx context.Context) error
	CallRPCRaw(subj string, request interface{}, optsFuncs ...RPCCallOptionFunc) ([]byte, error)
	CallRPC(subj string, request any, response any, optsFuncs ...RPCCallOptionFunc) error
}

type natsRPCImpl struct {
	nc     *nats.Conn
	routes map[string]*Route

	opts    NatsRPCOptions
	running bool
	ctx     context.Context
}

func New(natsConn *nats.Conn, options ...OptionFunc) NatsRPC {
	h := &natsRPCImpl{
		nc:     natsConn,
		routes: make(map[string]*Route),
		opts:   GetDefaultOptions(),
	}
	for _, v := range options {
		v(&h.opts)
	}
	return h
}

func (h *natsRPCImpl) AddRPC(method string, handler RPCHandleFunc, optsFuncs ...HandlerOptionFunc) {
	if h.ctx != nil {
		panic("cannot add RPC handler after StartWithContext called")
	}
	opts := h.opts.RPCHandlerOpts
	for _, o := range optsFuncs {
		o(&opts)
	}

	var path string
	if len(h.opts.BaseName) == 0 {
		path = method
	} else {
		path = fmt.Sprintf("%s.%s", h.opts.BaseName, method)
	}

	route := &Route{
		Path:    path,
		Handler: handler,
		Opts:    opts,
	}
	if _, ex := h.routes[route.Path]; ex {
		panic(fmt.Errorf("route %s already exists", route.Path))
	}
	h.routes[route.Path] = route
}

func (h *natsRPCImpl) StartWithContext(ctx context.Context) error {
	if len(h.routes) == 0 {
		panic("StartWithContext called, but no RPC handlers added")
	}
	h.ctx = ctx
	go func() {
		<-ctx.Done()
		for _, route := range h.routes {
			if err := route.Subscripion.Unsubscribe(); err != nil {
				log.Printf("failed to unsubscribe: %v", err)
			}
		}
	}()

	var err error
	for _, route := range h.routes {
		route.Subscripion, err = h.nc.Subscribe(route.Path, func(msg *nats.Msg) {
			// Create context
			ctx, cancell := context.WithTimeout(h.ctx, route.Opts.HandlerTimeout)
			defer cancell()
			// Create custom context
			nCtx := NewNatsRPCContext(ctx, msg, route)

			done := make(chan error, 1)
			// Call handler
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						done <- r.(error)
					}
				}()
				done <- route.Handler(nCtx)
			}()

			var doneErr error
			select {
			case err = <-done:
				doneErr = err
			case <-ctx.Done():
				doneErr = ctx.Err()
			}

			if !nCtx.ResponseWritten() {
				if doneErr != nil {
					err = nCtx.mustError(fmt.Sprintf("error in handler: %s", doneErr.Error()))
				} else {
					err = nCtx.mustError("no response written")
				}

				if err != nil {
					log.Printf("failed to respond to client: %s", err.Error())
				}
			}

		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (h *natsRPCImpl) CallRPCRaw(subj string, request interface{}, optsFuncs ...RPCCallOptionFunc) ([]byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	opts := h.opts.RPCCallOpts
	for _, o := range optsFuncs {
		o(&opts)
	}

	msg, err := h.nc.Request(subj, data, opts.Timeout)
	if err != nil {
		return nil, err
	}

	return msg.Data, nil
}

func (h *natsRPCImpl) CallRPC(subj string, request any, response any, optsFuncs ...RPCCallOptionFunc) error {
	data, err := h.CallRPCRaw(subj, request, optsFuncs...)
	if err != nil {
		return err
	}

	responseWrap := natsRPCResponse[any]{
		Data: response,
	}

	err = json.Unmarshal(data, &responseWrap)
	if err != nil {
		return err
	}

	if responseWrap.Error != nil {
		return fmt.Errorf("error in %s handler: %s", subj, responseWrap.Error.Message)
	}
	return nil
}

// DTO

type NatsError struct {
	Message string `json:"m"`
}

type natsRPCResponse[T any] struct {
	Data  T `json:"d"`
	Error *NatsError
}

type natsRPCRequest[T any] struct {
	Data T `json:"d"`
}
