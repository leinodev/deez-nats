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

func NewEventRouter(nc *nats.Conn) events.CoreNatsEvents {
	e := events.NewCoreEvents(nc,
		events.WithCoreQueueGroup("entity-queue"),
	)

	e.Use(eventLoggingMiddleware)

	e.AddEventHandler("entity.created", entityCreatedHandler,
		events.WithCoreHandlerQueue("entity-created-queue"),
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

func eventLoggingMiddleware(next events.HandlerFunc[*nats.Msg, nats.AckOpt]) events.HandlerFunc[*nats.Msg, nats.AckOpt] {
	return func(ctx events.EventContext[*nats.Msg, nats.AckOpt]) error {
		fmt.Printf("received event %s\n", ctx.Message().Subject)
		return next(ctx)
	}
}

func entityCreatedHandler(ctx events.EventContext[*nats.Msg, nats.AckOpt]) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity created: %#v\n", payload)
	return nil
}

func entityUpdatedHandler(ctx events.EventContext[*nats.Msg, nats.AckOpt]) error {
	var payload protocoljson.EntityEvent

	if err := ctx.Event(&payload); err != nil {
		return err
	}

	fmt.Printf("entity updated: %#v\n", payload)
	return nil
}
