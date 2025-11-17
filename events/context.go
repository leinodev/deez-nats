package events

import (
	"context"

	"github.com/leinodev/deez-nats/internal/ctx"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type coreEventContextImpl struct {
	*ctx.Base
}

func newCoreEventContext(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller) EventContext {
	return &coreEventContextImpl{
		Base: ctx.NewBase(parent, msg, m),
	}
}

// Event specific
func (c *coreEventContextImpl) Event(data any) error {
	return c.Unmarshal(data)
}
func (c *coreEventContextImpl) Ack(opts ...nats.AckOpt) error {
	return c.Message().Ack(opts...)
}
func (c *coreEventContextImpl) Nak(opts ...nats.AckOpt) error {
	return c.Message().Nak(opts...)
}
func (c *coreEventContextImpl) Term(opts ...nats.AckOpt) error {
	return c.Message().Term(opts...)
}
func (c *coreEventContextImpl) InProgress(opts ...nats.AckOpt) error {
	return c.Message().InProgress(opts...)
}

type jsEventContextImpl struct {
	*ctx.Base
	jsMsg jetstream.Msg
}

func newJsEventContext(parent context.Context, msg jetstream.Msg, m marshaller.PayloadMarshaller) EventContext {
	return &jsEventContextImpl{
		Base:  ctx.NewBase(parent, nil, m),
		jsMsg: msg,
	}
}

// Event specific
func (c *jsEventContextImpl) Unmarshal(data any) error {
	m := c.Marshaller()
	if c.jsMsg == nil || m == nil {
		return nil
	}

	return m.Unmarshall(c.jsMsg.Data(), &marshaller.MarshalObject{
		Data: data,
	})
}
func (c *jsEventContextImpl) Event(data any) error {
	return c.Unmarshal(data)
}
func (c *jsEventContextImpl) Ack(opts ...nats.AckOpt) error {
	return c.jsMsg.Ack()
}
func (c *jsEventContextImpl) Nak(opts ...nats.AckOpt) error {
	return c.jsMsg.Nak()
}
func (c *jsEventContextImpl) Term(opts ...nats.AckOpt) error {
	return c.jsMsg.Term()
}
func (c *jsEventContextImpl) InProgress(opts ...nats.AckOpt) error {
	return c.jsMsg.InProgress()
}
