package main

import (
	"context"
	"errors"
	"log"
	"simple/protocoljson"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/rpc"
)

func main() {
	// Connect to nats cluster
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	eventService := NewEventRouter(nc)
	rpcService := NewRpcRouter(nc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := eventService.StartWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("events stopped with error: %v", err)
		}
	}()

	go func() {
		if err := rpcService.StartWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("rpc stopped with error: %v", err)
		}
	}()

	// Emit example events
	err = eventService.Emit(ctx, "entity.created", protocoljson.EntityEvent{ID: "42", Name: "example"}, nil)
	if err != nil {
		log.Printf("publish entity.created: %v", err)
	}

	err = eventService.Emit(ctx, "entity.updated", protocoljson.EntityEvent{ID: "42", Name: "example-renamed"}, nil)
	if err != nil {
		log.Printf("publish entity.updated: %v", err)
	}

	var response protocoljson.MyResponse
	if err := rpcService.CallRPC("test", protocoljson.MyRequst{UserID: "42"}, &response, rpc.CallOptions{}); err != nil {
		log.Printf("rpc call test: %v", err)
	}

	time.Sleep(2 * time.Second)
}
