package newevents

import (
	"context"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type coreContextImpl struct {
	msg *nats.Msg
	ctx context.Context
	m   marshaller.PayloadMarshaller
}

func newCoreContext(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller) EventContext[*nats.Msg, nats.AckOpt] {
	if parent == nil {
		parent = context.Background()
	}

	return &coreContextImpl{
		ctx: parent,
		msg: msg,
		m:   m,
	}
}

// Context inherited
func (c *coreContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *coreContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *coreContextImpl) Err() error {
	return c.ctx.Err()
}
func (c *coreContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// Common methods
func (c *coreContextImpl) Headers() nats.Header {
	return c.msg.Header
}
func (c *coreContextImpl) Message() *nats.Msg {
	return c.msg
}
func (c *coreContextImpl) Event(data any) error {
	return c.m.Unmarshall(c.msg.Data, &marshaller.MarshalObject{
		Data: data,
	})
}

// Ack inherited
func (c *coreContextImpl) Ack(opts ...nats.AckOpt) error {
	return c.msg.Ack(opts...)
}
func (c *coreContextImpl) Nak(opts ...nats.AckOpt) error {
	return c.msg.Nak(opts...)
}
func (c *coreContextImpl) Term(opts ...nats.AckOpt) error {
	return c.msg.Term(opts...)
}
func (c *coreContextImpl) InProgress(opts ...nats.AckOpt) error {
	return c.msg.InProgress(opts...)
}
