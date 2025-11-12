package events

import (
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

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
