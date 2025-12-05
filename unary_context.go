package natsrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type UnaryContext interface {
	context.Context
	Request(data any) error

	Ok(data any) error

	RequestHeaders() nats.Header
	Headers() nats.Header

	responseWritten() bool

	writeError(err error) error

	run(handler RpcUnaryHandleFunc)
}

type unaryContextImpl struct {
	root   NatsRPC
	ctx    context.Context
	cancel context.CancelFunc
	msg    *nats.Msg

	handlerOptions  HandlerOptions
	responseHeaders nats.Header
	respWr          bool
}

func newRpcContext(root NatsRPC, ctx context.Context, msg *nats.Msg, handlerOptions HandlerOptions) UnaryContext {
	ctx, cancel := context.WithTimeout(ctx, handlerOptions.Timeout)

	return &unaryContextImpl{
		root:            root,
		ctx:             ctx,
		cancel:          cancel,
		msg:             msg,
		handlerOptions:  handlerOptions,
		responseHeaders: nats.Header{},
		respWr:          false,
	}
}

// Context methods
func (c *unaryContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *unaryContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *unaryContextImpl) Err() error {
	return c.ctx.Err()
}
func (c *unaryContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

func (c *unaryContextImpl) Request(data any) error {
	encoding := c.RequestHeaders().Get("encoding")
	encoder, ok := c.root.GetEncoding(encoding)
	if !ok {
		return fmt.Errorf("not found suitable encoding with name %s", encoding)
	}
	return encoder.Unmarshall(c.msg.Data, &marshaller.MarshalObject{
		Data: data,
	})
}
func (c *unaryContextImpl) Ok(data any) error {
	if c.respWr {
		return ErrResponseAlreadyWritten
	}

	obj := &marshaller.MarshalObject{
		Data: data,
	}

	if err := c.respond(obj); err != nil {
		return err
	}

	c.respWr = true
	return nil
}

func (c *unaryContextImpl) RequestHeaders() nats.Header {
	return c.msg.Header
}
func (c *unaryContextImpl) Headers() nats.Header {
	return c.responseHeaders
}

func (c *unaryContextImpl) responseWritten() bool {
	return c.respWr
}
func (c *unaryContextImpl) writeError(werr error) error {
	if werr == nil {
		return ErrEmptyError
	}
	if c.respWr {
		return ErrResponseAlreadyWritten
	}
	errCode := errCode[werr]
	err := c.respond(&marshaller.MarshalObject{
		Err: &marshaller.Error{
			Text: werr.Error(),
			Code: errCode,
		},
	})
	if err != nil {
		return werr
	}
	c.respWr = true
	return nil
}
func (c *unaryContextImpl) respond(obj *marshaller.MarshalObject) error {
	defer c.cancel()

	encoding := c.RequestHeaders().Get("encoding")
	encoder, ok := c.root.GetEncoding(encoding)
	if !ok {
		return fmt.Errorf("not found suitable encoding with name %s", encoding)
	}

	data, err := encoder.Marshall(obj)
	if err != nil {
		return err
	}
	return c.msg.RespondMsg(&nats.Msg{
		Subject: c.msg.Reply,
		Data:    data,
		Header:  c.responseHeaders,
	})
}

func (c *unaryContextImpl) run(handler RpcUnaryHandleFunc) {
	err := handler(c)
	if c.responseWritten() {
		_ = c.msg.Ack()
		return
	}

	if err == nil {
		// if no error, repond with empty data
		err = c.Ok(nil)
	}
	if err != nil {
		// Got an error, respond with error
		_ = c.writeError(err)
	}
}
