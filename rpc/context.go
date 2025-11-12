package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var errResponseAlreadyWritten = errors.New("rpc response already written")

func newRpcContext(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller, writer func(*marshaller.MarshalObject, nats.Header) error) RPCContext {
	if parent == nil {
		parent = context.Background()
	}
	if m == nil {
		m = marshaller.DefaultJsonMarshaller
	}

	return &rpcContextImpl{
		ctx:             parent,
		msg:             msg,
		marshaller:      m,
		writer:          writer,
		requestHeaders:  cloneHeader(msg.Header),
		responseHeaders: nil,
	}
}

type rpcContextImpl struct {
	ctx context.Context

	msg        *nats.Msg
	marshaller marshaller.PayloadMarshaller

	writer func(*marshaller.MarshalObject, nats.Header) error

	requestHeaders  nats.Header
	responseHeaders nats.Header

	responseWritten bool
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

// Rpc methods
func (c *rpcContextImpl) Request(data any) error {
	if c.msg == nil {
		return errors.New("nil rpc message")
	}

	payload := &marshaller.MarshalObject{
		Data: data,
	}

	if err := c.marshaller.Unmarshall(c.msg.Data, payload); err != nil {
		return err
	}

	if payload.Error != "" {
		return errors.New(payload.Error)
	}

	return nil
}

func (c *rpcContextImpl) Ok(data any) error {
	if c.responseWritten {
		return errResponseAlreadyWritten
	}

	if c.writer == nil {
		return errors.New("rpc writer is nil")
	}

	obj := &marshaller.MarshalObject{
		Data: data,
	}

	if err := c.writer(obj, cloneHeader(c.Headers())); err != nil {
		return err
	}

	c.responseWritten = true
	return nil
}

func (c *rpcContextImpl) RequestHeaders() nats.Header {
	return c.requestHeaders
}

func (c *rpcContextImpl) Headers() nats.Header {
	if c.responseHeaders == nil {
		c.responseHeaders = nats.Header{}
	}
	return c.responseHeaders
}

func (c *rpcContextImpl) writeError(err error) error {
	if err == nil {
		return nil
	}
	if c.responseWritten {
		return errResponseAlreadyWritten
	}
	if c.writer == nil {
		return errors.New("rpc writer is nil")
	}
	obj := &marshaller.MarshalObject{
		Error: err.Error(),
	}

	if werr := c.writer(obj, cloneHeader(c.Headers())); werr != nil {
		return werr
	}

	c.responseWritten = true
	return nil
}

func (c *rpcContextImpl) hasResponse() bool {
	return c.responseWritten
}

func cloneHeader(h nats.Header) nats.Header {
	if len(h) == 0 {
		return nil
	}
	copy := nats.Header{}
	for k, values := range h {
		copy[k] = append([]string(nil), values...)
	}
	return copy
}
