package natsrpcgo

import (
	"time"

	"github.com/TexHik620953/natsrpc-go/marshaller"
	"github.com/nats-io/nats.go"
)

type HandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
}

type CallOptions struct {
	Marshaller marshaller.PayloadMarshaller
}

type EventHandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Queue      string
	JetStream  JetStreamEventOptions
}

type JetStreamEventOptions struct {
	Enabled          bool
	AutoAck          bool
	Pull             bool
	PullBatch        int
	PullExpire       time.Duration
	Durable          string
	DeliverGroup     string
	SubjectTransform func(subject string) string
	SubscribeOptions []nats.SubOpt
}

type EventPublishOptions struct {
	Marshaller marshaller.PayloadMarshaller
	Headers    map[string]string
	JetStream  []nats.PubOpt
}
