package events

import (
	"context"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

func newEventContext(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller) EventContext {
	if parent == nil {
		parent = context.Background()
	}

	return &eventContextImpl{
		ctx:        parent,
		msg:        msg,
		marshaller: m,
	}
}

type eventContextImpl struct {
	ctx        context.Context
	msg        *nats.Msg
	marshaller marshaller.PayloadMarshaller
}

// Context inherited
func (c *eventContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *eventContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *eventContextImpl) Err() error {
	return c.ctx.Err()
}

func (c *eventContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// Event specific
func (c *eventContextImpl) Event(data any) error {
	if c.msg == nil {
		return nil
	}

	return c.marshaller.Unmarshall(c.msg.Data, &marshaller.MarshalObject{
		Data: data,
	})
}

func (c *eventContextImpl) Headers() nats.Header {
	if c.msg == nil {
		return nil
	}
	return c.msg.Header
}

func (c *eventContextImpl) Message() *nats.Msg {
	return c.msg
}

func (c *eventContextImpl) Ack(opts ...nats.AckOpt) error {
	if c.msg == nil {
		return nil
	}

	return c.msg.Ack(opts...)
}

func (c *eventContextImpl) Nak(opts ...nats.AckOpt) error {
	if c.msg == nil {
		return nil
	}

	return c.msg.Nak(opts...)
}

func (c *eventContextImpl) Term(opts ...nats.AckOpt) error {
	if c.msg == nil {
		return nil
	}

	return c.msg.Term(opts...)
}

func (c *eventContextImpl) InProgress(opts ...nats.AckOpt) error {
	if c.msg == nil {
		return nil
	}

	return c.msg.InProgress(opts...)
}
