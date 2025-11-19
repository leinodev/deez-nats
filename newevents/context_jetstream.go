package newevents

import (
	"context"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type jetstreamContextImpl struct {
	msg jetstream.Msg
	ctx context.Context
	m   marshaller.PayloadMarshaller
}

func newJetstreamContext(parent context.Context, msg jetstream.Msg, m marshaller.PayloadMarshaller) EventContext[jetstream.Msg, any] {
	if parent == nil {
		parent = context.Background()
	}

	return &jetstreamContextImpl{
		ctx: parent,
		msg: msg,
		m:   m,
	}
}

// Context inherited
func (c *jetstreamContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *jetstreamContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *jetstreamContextImpl) Err() error {
	return c.ctx.Err()
}
func (c *jetstreamContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// Common methods
func (c *jetstreamContextImpl) Headers() nats.Header {
	return c.msg.Headers()
}
func (c *jetstreamContextImpl) Message() jetstream.Msg {
	return c.msg
}
func (c *jetstreamContextImpl) Event(data any) error {
	return c.m.Unmarshall(c.msg.Data(), &marshaller.MarshalObject{
		Data: data,
	})
}

// Ack inherited
func (c *jetstreamContextImpl) Ack(opts ...any) error {
	return c.msg.Ack()
}
func (c *jetstreamContextImpl) Nak(opts ...any) error {
	return c.msg.Nak()
}
func (c *jetstreamContextImpl) Term(opts ...any) error {
	return c.msg.Term()
}
func (c *jetstreamContextImpl) InProgress(opts ...any) error {
	return c.msg.InProgress()
}
