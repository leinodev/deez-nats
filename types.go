package natsrpc

import (
	"context"

	"github.com/nats-io/nats.go"
)

type RpcUnaryHandleFunc func(c UnaryContext) error
type RpcUnaryMiddlewareFunc func(next RpcUnaryHandleFunc) RpcUnaryHandleFunc

type NatsConn interface {
	RequestMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
	ChanSubscribe(subj string, ch chan *nats.Msg) (*nats.Subscription, error)
}
