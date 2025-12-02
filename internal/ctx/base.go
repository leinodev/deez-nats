package ctx

import (
	"context"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type Base struct {
	ctx        context.Context
	msg        *nats.Msg
	marshaller marshaller.PayloadMarshaller
}

func NewBase(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller) *Base {
	if parent == nil {
		parent = context.Background()
	}

	return &Base{
		ctx:        parent,
		msg:        msg,
		marshaller: m,
	}
}

// Context methods
func (c *Base) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *Base) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *Base) Err() error {
	return c.ctx.Err()
}
func (c *Base) Value(key any) any {
	return c.ctx.Value(key)
}

// Common methods
func (c *Base) Headers() nats.Header {
	if c.msg == nil {
		return nil
	}
	return c.msg.Header
}
func (c *Base) Message() *nats.Msg {
	return c.msg
}
func (c *Base) Unmarshal(data any) error {
	if c.msg == nil || c.marshaller == nil {
		return nil
	}

	return c.marshaller.Unmarshall(c.msg.Data, &marshaller.MarshalObject{
		Data: data,
	})
}
func (c *Base) Context() context.Context {
	return c.ctx
}
func (c *Base) Marshaller() marshaller.PayloadMarshaller {
	return c.marshaller
}
