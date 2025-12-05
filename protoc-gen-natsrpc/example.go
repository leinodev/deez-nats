package srakaName


import (
	"context"
	"log"
	"time"

	"google.golang.org/protobuf/proto"
	"github.com/nats-io/nats.go"
	"github.com/leinodev/deez-nats"
)



type SrakaHandler struct {
	ctx     context.Context
	workers *nrpc.WorkerPool
	nc      nrpc.NatsConn
	server  Server

	encodings []string
}

func New{{.GetName}}Handler(ctx context.Context, nc nrpc.NatsConn, s {{.GetName}}Server) *{{.GetName}}Handler {
	return &{{.GetName}}Handler{
		ctx:    ctx,
		nc:     nc,
		server: s,

		encodings: []string{"protobuf"},
	}
}