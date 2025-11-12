package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var (
	ErrResponseAlreadyWritten = errors.New("rpc response already written")
	ErrEmptyError             = errors.New("cannot respond with empty error")
)

func newRpcContext(parent context.Context, msg *nats.Msg, handlerOptions HandlerOptions) RPCContext {
	return &rpcContextImpl{
		ctx:             parent,
		msg:             msg,
		handlerOptions:  handlerOptions,
		responseHeaders: nats.Header{},
		respWr:          false,
	}
}

type rpcContextImpl struct {
	ctx context.Context

	msg            *nats.Msg
	handlerOptions HandlerOptions

	responseHeaders nats.Header

	respWr bool
}

// Context inherited
func (c *rpcContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *rpcContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *rpcContextImpl) Err() error {
	return c.ctx.Err()
}
func (c *rpcContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// Rpc inherited methods
func (c *rpcContextImpl) Request(data any) error {
	err := c.handlerOptions.Marshaller.Unmarshall(c.msg.Data, &marshaller.MarshalObject{
		Data: data,
	})
	if err != nil {
		return err
	}

	return nil
}
func (c *rpcContextImpl) Ok(data any) error {
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
func (c *rpcContextImpl) RequestHeaders() nats.Header {
	return c.msg.Header
}
func (c *rpcContextImpl) Headers() nats.Header {
	return c.responseHeaders
}
func (c *rpcContextImpl) responseWritten() bool {
	return c.respWr
}
func (c *rpcContextImpl) writeError(werr error) error {
	if werr == nil {
		return ErrEmptyError
	}
	if c.respWr {
		return ErrResponseAlreadyWritten
	}

	err := c.respond(&marshaller.MarshalObject{
		Error: werr.Error(),
	})
	if err != nil {
		return werr
	}
	c.respWr = true
	return nil
}

func (c *rpcContextImpl) respond(obj *marshaller.MarshalObject) error {
	data, err := c.handlerOptions.Marshaller.Marshall(obj)
	if err != nil {
		return err
	}
	return c.msg.RespondMsg(&nats.Msg{
		Subject: c.msg.Reply,
		Data:    data,
		Header:  c.responseHeaders,
	})
}
