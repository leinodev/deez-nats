package natsrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type NatsRPC interface {
	RPCRouter

	UnaryCall(ctx context.Context, method string, request any, response any, opts ...CallOption) error

	StartWithContext(ctx context.Context) error
	Shutdown(ctx context.Context) error

	GetEncoding(enc string) (marshaller.PayloadMarshaller, bool)
	RegisterEncoding(name string, enc marshaller.PayloadMarshaller)
}

type natsRpcImpl struct {
	NatsRPC

	nc         NatsConn
	options    NatsRPCOptions
	rootRouter RPCRouter

	encodings map[string]marshaller.PayloadMarshaller

	wokerPool *workerPool
}

// TODO: Add logger
func New(nc NatsConn, opts ...NatsRPCOption) NatsRPC {
	options := NewRPCOptions(opts...)

	marshallers := map[string]marshaller.PayloadMarshaller{
		"json":  marshaller.DefaultJsonMarshaller,
		"proto": marshaller.DefaultProtoMarshaller,
	}

	return &natsRpcImpl{
		nc:         nc,
		options:    options,
		rootRouter: newRouter(options.BaseRoute, options.DefaultHandlerOptions),
		encodings:  marshallers,
	}
}

// Encodings
func (r *natsRpcImpl) GetEncoding(enc string) (marshaller.PayloadMarshaller, bool) {
	e, ok := r.encodings[enc]
	return e, ok
}
func (r *natsRpcImpl) RegisterEncoding(name string, enc marshaller.PayloadMarshaller) {
	r.encodings[name] = enc
}

// Router inherited
func (r *natsRpcImpl) Use(middlewares ...RpcUnaryMiddlewareFunc) {
	r.rootRouter.Use(middlewares...)
}
func (r *natsRpcImpl) AddUnaryHandler(method string, handler RpcUnaryHandleFunc, opts ...HandlerOption) {
	r.rootRouter.AddUnaryHandler(method, handler, opts...)
}
func (r *natsRpcImpl) Group(group string) RPCRouter {
	return r.rootRouter.Group(group)
}

// Rpc methods
func (r *natsRpcImpl) UnaryCall(ctx context.Context, subj string, request any, response any, opts ...CallOption) error {
	if subj == "" {
		return fmt.Errorf("%w: empty subject", ErrInvalidSubject)
	}

	callOpts := r.options.DefaultCallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	if callOpts.Encoding == "" {
		callOpts.Encoding = r.options.DefaultCallOptions.Encoding
	}
	encoder, ok := r.GetEncoding(callOpts.Encoding)
	if !ok {
		callOpts.Encoding = "json"
		encoder = marshaller.DefaultJsonMarshaller
	}

	headers := make(nats.Header)

	for k, v := range callOpts.Headers {
		headers[k] = v
	}
	headers.Set("encoding", callOpts.Encoding)

	if !ok {
		return fmt.Errorf("not found suitable encoding with name %s", callOpts.Encoding)
	}

	payload, err := encoder.Marshall(&marshaller.MarshalObject{
		Data: request,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := r.nc.RequestMsgWithContext(ctx, &nats.Msg{
		Subject: subj,
		Data:    payload,
		Header:  headers,
	})
	if err != nil {
		return err
	}

	respObj := &marshaller.MarshalObject{
		Data: response,
	}

	if err := encoder.Unmarshall(msg.Data, respObj); err != nil {
		return fmt.Errorf("unmarshall response: %w", err)
	}

	if respObj.Err != nil {
		nerr, ok := codeErr[respObj.Err.Code]
		if ok {
			return fmt.Errorf("%w: %s", nerr, respObj.Err.Text)
		} else {
			return fmt.Errorf("%s", respObj.Err.Text)
		}
	}
	return nil
}

// Controll methods
func (r *natsRpcImpl) StartWithContext(ctx context.Context) error {
	// Create worker pool
	r.wokerPool = newWorkerPool(r, ctx, r.options.WorkersCount, r.options.MsgPoolSize)

	// Build routes handlers
	for _, route := range r.rootRouter.dfs() {
		r.wokerPool.routes[route.method] = &route
	}

	var err error
	for subj, route := range r.wokerPool.routes {
		route.sub, err = r.nc.ChanSubscribe(subj, r.wokerPool.inChan)
		if err != nil {
			break
		}
	}
	if err != nil {
		_ = r.Shutdown(ctx)
		return fmt.Errorf("failed to subscribe: %w", err)
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
	return r.wokerPool.Stop(ctx)
}
