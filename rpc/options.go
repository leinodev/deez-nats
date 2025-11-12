package rpc

import "github.com/leinodev/deez-nats/marshaller"

type HandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
}

type CallOptions struct {
	Marshaller marshaller.PayloadMarshaller
}
