package main

import (
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/events"
	"github.com/leinodev/deez-nats/rpc"
)

func NewRpcRouter(nc *nats.Conn) rpc.NatsRPC {
	service := rpc.NewNatsRPC(nc)

	rpc.AddTypedJsonRPCHandler(service, "user.get", func(ctx rpc.RPCContext, request GetUserRequest) (GetUserResponse, error) {
		if request.ID == "" {
			return GetUserResponse{}, fmt.Errorf("empty id")
		}

		ctx.Headers().Set("X-Handler", "typed-json")

		return GetUserResponse{
			ID:   request.ID,
			Name: "Typed Example",
		}, nil
	})

	rpc.AddTypedJsonRPCHandler(service, "user.update", func(ctx rpc.RPCContext, request UpdateUserRequest) (UpdateUserResponse, error) {
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

func NewEventRouter(nc *nats.Conn) events.CoreNatsEvents {
	service := events.NewCoreEvents(nc,
		events.WithCoreQueueGroup("user-queue"),
	)

	events.AddTypedCoreJsonEventHandler(service, "user.created", func(ctx events.EventContext[*nats.Msg, nats.AckOpt], payload UserCreatedEvent) error {
		fmt.Printf("user created: %#v\n", payload)
		return nil
	})

	events.AddTypedCoreJsonEventHandler(service, "user.renamed", func(ctx events.EventContext[*nats.Msg, nats.AckOpt], payload UserRenamedEvent) error {
		fmt.Printf("user renamed: %#v\n", payload)
		return nil
	})

	return service
}
