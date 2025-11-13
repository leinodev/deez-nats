package main

import (
	"fmt"
	"simple/protocoljson"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/events"
	"github.com/leinodev/deez-nats/rpc"
)

func NewRpcRouter(nc *nats.Conn) rpc.NatsRPC {
	r := rpc.NewNatsRPC(nc, nil)

	r.AddRPCHandler("test", testHandler, nil)
	r.AddRPCHandler("sraka", test1Handler, nil)

	return r
}

func NewEventRouter(nc *nats.Conn) events.Events {
	e := events.NewEvents(nc,
		events.WithEventDefaultHandlerOptions(events.EventHandlerOptions{
			Queue: "",
			JetStream: events.JetStreamEventOptions{
				Enabled: true,
				AutoAck: true,
			},
		}),
	)

	e.Use(eventLoggingMiddleware)

	e.AddEventHandler("entity.created", entityCreatedHandler, &events.EventHandlerOptions{
		JetStream: events.JetStreamEventOptions{
			Enabled:      true,
			AutoAck:      true,
			Durable:      "entity-created-consumer",
			DeliverGroup: "entity-events",
		},
	})

	e.AddEventHandler("entity.updated", entityUpdatedHandler, nil)

	return e
}

func testHandler(c rpc.RPCContext) error {
	return nil
}

func test1Handler(c rpc.RPCContext) error {
	var msg protocoljson.MyRequst
	err := c.Request(&msg)
	if err != nil {
		return err
	}
	return nil
}

func eventLoggingMiddleware(next events.EventHandleFunc) events.EventHandleFunc {
	return func(ctx events.EventContext) error {
		fmt.Printf("received event %s\n", ctx.Message().Subject)
		return next(ctx)
	}
}

func entityCreatedHandler(ctx events.EventContext) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity created: %#v\n", payload)
	return nil
}

func entityUpdatedHandler(ctx events.EventContext) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity updated: %#v\n", payload)
	return nil
}
