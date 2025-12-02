package main

import (
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/natsevents"
	"github.com/leinodev/deez-nats/natsrpc"
)

func NewRpcRouter(nc *nats.Conn) natsrpc.NatsRPC {
	service := natsrpc.New(nc)

	natsrpc.AddTypedJsonRPCHandler(service, "user.get", func(ctx natsrpc.RPCContext, request GetUserRequest) (GetUserResponse, error) {
		if request.ID == "" {
			return GetUserResponse{}, fmt.Errorf("empty id")
		}

		ctx.Headers().Set("X-Handler", "typed-json")

		return GetUserResponse{
			ID:   request.ID,
			Name: "Typed Example",
		}, nil
	})

	natsrpc.AddTypedJsonRPCHandler(service, "user.update", func(ctx natsrpc.RPCContext, request UpdateUserRequest) (UpdateUserResponse, error) {
		if request.ID == "" {
			return UpdateUserResponse{}, fmt.Errorf("empty id")
		}

		ctx.Headers().Set("X-Handled-By", "typed-with-marshaller")

		return UpdateUserResponse{
			Updated: true,
		}, nil
	})

	return service
}

func NewEventRouter(nc *nats.Conn) natsevents.CoreNatsEvents {
	service := natsevents.New(nc,
		natsevents.WithCoreQueueGroup("user-queue"),
	)

	natsevents.AddTypedCoreJsonEventHandler(service, "user.created", func(ctx natsevents.EventContext[*nats.Msg, nats.AckOpt], payload UserCreatedEvent) error {
		fmt.Printf("user created: %#v\n", payload)
		return nil
	})

	natsevents.AddTypedCoreJsonEventHandler(service, "user.renamed", func(ctx natsevents.EventContext[*nats.Msg, nats.AckOpt], payload UserRenamedEvent) error {
		fmt.Printf("user renamed: %#v\n", payload)
		return nil
	})

	return service
}
