package natsrpcgo

import "github.com/TexHik620953/natsrpc-go/marshaller"

type HandlerOptions struct {
	Marshaller marshaller.PayloadMarshaller
}

type CallOptions struct {
	Marshaller marshaller.PayloadMarshaller
}
