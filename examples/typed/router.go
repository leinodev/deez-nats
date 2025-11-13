package main

import (
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/events"
	"github.com/leinodev/deez-nats/rpc"
)

func NewRpcRouter(nc *nats.Conn) rpc.NatsRPC {
	service := rpc.NewNatsRPC(nc, nil)

	rpc.AddTypedJsonRPCHandler(service, "user.get", func(ctx rpc.RPCContext, request GetUserRequest) (GetUserResponse, error) {
		if request.ID == "" {
			return GetUserResponse{}, fmt.Errorf("empty id")
		}

		ctx.Headers().Set("X-Handler", "typed-json")

		return GetUserResponse{
			ID:   request.ID,
			Name: "Typed Example",
		}, nil
	}, nil)

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

func NewEventRouter(nc *nats.Conn) events.NatsEvents {
	opts := events.NewEventsOptionsBuilder().
		WithDefaultHandlerOptions(func(b *events.EventHandlerOptionsBuilder) {
			b.WithJetStream(func(jsb *events.JetStreamEventOptionsBuilder) {
				jsb.Enabled().WithAutoAck(true)
			})
		}).
		Build()
	service := events.NewNatsEvents(nc, &opts)

	events.AddTypedEventHandler(service, "user.created", func(ctx events.EventContext, payload UserCreatedEvent) error {
		fmt.Printf("user created via typed handler: %#v\n", payload)
		return nil
	}, nil)

	events.AddTypedJsonEventHandler(service, "user.renamed", func(ctx events.EventContext, payload UserRenamedEvent) error {
		fmt.Printf("user renamed via typed handler with marshaller: %#v\n", payload)
		return nil
	})

	return service
}
