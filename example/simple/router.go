package main

import (
	"fmt"
	"simple/protocoljson"

	natsrpcgo "github.com/TexHik620953/natsrpc-go"
	"github.com/nats-io/nats.go"
)

func NewRpcRouter(nc *nats.Conn) natsrpcgo.NatsRPC {
	rpc := natsrpcgo.NewNatsRPC(nc)

	rpc.AddRPCHandler("test", testHandler, nil)
	rpc.AddRPCHandler("sraka", test1Handler, nil)

	return rpc
}

func NewEventRouter(nc *nats.Conn) natsrpcgo.Events {
	events := natsrpcgo.NewEvents(nc,
		natsrpcgo.WithEventDefaultHandlerOptions(natsrpcgo.EventHandlerOptions{
			Queue: "",
			JetStream: natsrpcgo.JetStreamEventOptions{
				Enabled: true,
				AutoAck: true,
			},
		}),
	)

	events.Use(eventLoggingMiddleware)

	events.AddEventHandler("entity.created", entityCreatedHandler, &natsrpcgo.EventHandlerOptions{
		JetStream: natsrpcgo.JetStreamEventOptions{
			Enabled:      true,
			AutoAck:      true,
			Durable:      "entity-created-consumer",
			DeliverGroup: "entity-events",
		},
	})

	events.AddEventHandler("entity.updated", entityUpdatedHandler, nil)

	return events
}

func testHandler(c natsrpcgo.RPCContext) error {
	return nil
}

func test1Handler(c natsrpcgo.RPCContext) error {
	var msg protocoljson.MyRequst
	err := c.Request(&msg)
	if err != nil {
		return err
	}
	return nil
}

func eventLoggingMiddleware(next natsrpcgo.EventHandleFunc) natsrpcgo.EventHandleFunc {
	return func(ctx natsrpcgo.EventContext) error {
		fmt.Printf("received event %s\n", ctx.Message().Subject)
		return next(ctx)
	}
}

func entityCreatedHandler(ctx natsrpcgo.EventContext) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity created: %#v\n", payload)
	return nil
}

func entityUpdatedHandler(ctx natsrpcgo.EventContext) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity updated: %#v\n", payload)
	return nil
}
