package natsrpcgo

import (
	"context"
	"time"
)

type RPCContext interface {
	context.Context
	Request(data any) error

	Ok(data any) error

	RequestHeaders() RHeaders
	Headers() RWHeaders
}

func newRpcContext() RPCContext {
	return &rpcContextImpl{
		ctx:             context.Background(),
		responseWritten: false,
	}
}

type rpcContextImpl struct {
	ctx             context.Context
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
	return nil
}
func (c *rpcContextImpl) Ok(data any) error {
	return nil
}
func (c *rpcContextImpl) RequestHeaders() RHeaders {
	return nil
}
func (c *rpcContextImpl) Headers() RWHeaders {
	return nil
}
