package events

import (
	"context"

	"github.com/leinodev/deez-nats/internal/ctx"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

func newEventContext(parent context.Context, msg *nats.Msg, m marshaller.PayloadMarshaller) EventContext {
	return &eventContextImpl{
		Base: ctx.NewBase(parent, msg, m),
	}
}

type eventContextImpl struct {
	*ctx.Base
}

// Event specific
func (c *eventContextImpl) Event(data any) error {
	return c.Unmarshal(data)
}
func (c *eventContextImpl) Ack(opts ...nats.AckOpt) error {
	return c.Message().Ack(opts...)
}
func (c *eventContextImpl) Nak(opts ...nats.AckOpt) error {
	return c.Message().Nak(opts...)
}
func (c *eventContextImpl) Term(opts ...nats.AckOpt) error {
	return c.Message().Term(opts...)
}
func (c *eventContextImpl) InProgress(opts ...nats.AckOpt) error {
	return c.Message().InProgress(opts...)
}
