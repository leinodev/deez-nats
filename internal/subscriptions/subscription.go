package subscriptions

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Core subscription
type coreSubImpl struct {
	sub *nats.Subscription
}

func NewCoreSub(sub *nats.Subscription) Subscription {
	return &coreSubImpl{sub: sub}
}

func (s *coreSubImpl) Drain() error {
	return s.sub.Drain()
}

func (s *coreSubImpl) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

// JetStream subscription
type jsSubImpl struct {
	ctx jetstream.ConsumeContext
}

func NewJsSub(ctx jetstream.ConsumeContext) Subscription {
	return &jsSubImpl{ctx: ctx}
}

func (s *jsSubImpl) Drain() error {
	s.ctx.Drain()
	return nil
}
func (s *jsSubImpl) Unsubscribe() error {
	s.ctx.Stop()
	return nil
}
