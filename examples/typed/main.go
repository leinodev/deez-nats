package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/leinodev/deez-nats/events"
	"github.com/leinodev/deez-nats/rpc"
)

func main() {
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

	emitExamples(ctx, eventService)
	callExamples(rpcService)

	time.Sleep(2 * time.Second)
}

func emitExamples(ctx context.Context, service events.Events) {
	if err := service.Emit(ctx, "user.created", UserCreatedEvent{ID: "42", Name: "Typed Example"}, nil); err != nil {
		log.Printf("emit user.created: %v", err)
	}

	if err := service.Emit(ctx, "user.renamed", UserRenamedEvent{ID: "42", NewName: "Typed Example v2"}, nil); err != nil {
		log.Printf("emit user.renamed: %v", err)
	}
}

func callExamples(service rpc.NatsRPC) {
	var getResp GetUserResponse
	if err := service.CallRPC("user.get", GetUserRequest{ID: "42"}, &getResp, rpc.CallOptions{}); err != nil {
		log.Printf("rpc call user.get: %v", err)
	} else {
		log.Printf("user.get response: %#v", getResp)
	}

	var updateResp UpdateUserResponse
	if err := service.CallRPC("user.update", UpdateUserRequest{ID: "42", Name: "Typed Example v2"}, &updateResp, rpc.CallOptions{}); err != nil {
		log.Printf("rpc call user.update: %v", err)
	} else {
		log.Printf("user.update response: %#v", updateResp)
	}
}
