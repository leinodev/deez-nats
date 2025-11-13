package main

import (
	"fmt"
	"simple/protocoljson"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/events"
	"github.com/leinodev/deez-nats/rpc"
)

func NewRpcRouter(nc *nats.Conn) rpc.NatsRPC {
	r := rpc.NewNatsRPC(nc)

	r.AddRPCHandler("test", testHandler)
	r.AddRPCHandler("sraka", test1Handler)

	return r
}

func NewEventRouter(nc *nats.Conn) events.NatsEvents {
	e := events.NewNatsEvents(nc,
		events.WithJetStream(true),
		events.WithAutoAck(true),
	)

	e.Use(eventLoggingMiddleware)

	e.AddEventHandler("entity.created", entityCreatedHandler,
		events.WithHandlerJetStream(
			events.WithJSEnabled(true),
			events.WithJSAutoAck(true),
			events.WithJSDurable("entity-created-consumer"),
			events.WithJSDeliverGroup("entity-events"),
		),
	)

	e.AddEventHandler("entity.updated", entityUpdatedHandler)

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
