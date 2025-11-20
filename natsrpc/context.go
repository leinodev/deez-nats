package natsrpc

import (
	"context"
	"errors"

	"github.com/leinodev/deez-nats/internal/ctx"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var (
	ErrResponseAlreadyWritten = errors.New("rpc response already written")
	ErrEmptyError             = errors.New("cannot respond with empty error")
)

func newRpcContext(parent context.Context, msg *nats.Msg, handlerOptions HandlerOptions) RPCContext {
	return &rpcContextImpl{
		Base:            ctx.NewBase(parent, msg, handlerOptions.Marshaller),
		handlerOptions:  handlerOptions,
		responseHeaders: nats.Header{},
		respWr:          false,
	}
}

type rpcContextImpl struct {
	*ctx.Base

	handlerOptions HandlerOptions

	responseHeaders nats.Header

	respWr bool
}

// Rpc inherited methods
func (c *rpcContextImpl) Request(data any) error {
	return c.Unmarshal(data)
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
	return c.Headers()
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
	data, err := c.Marshaller().Marshall(obj)
	if err != nil {
		return err
	}
	return c.Message().RespondMsg(&nats.Msg{
		Subject: c.Message().Reply,
		Data:    data,
		Header:  c.responseHeaders,
	})
}
