package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
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
	callExamples(ctx, rpcService)

	// Graceful shutdown: wait for shutdown signal (SIGINT or SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Received shutdown signal, starting graceful shutdown...")

	// Cancel context to stop processing new messages
	cancel()

	// Perform graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := eventService.Shutdown(shutdownCtx); err != nil {
		log.Printf("events service shutdown error: %v", err)
	} else {
		log.Println("events service shutdown completed")
	}

	if err := rpcService.Shutdown(shutdownCtx); err != nil {
		log.Printf("rpc service shutdown error: %v", err)
	} else {
		log.Println("rpc service shutdown completed")
	}

	log.Println("Graceful shutdown completed")
}

func emitExamples(ctx context.Context, service events.NatsEvents) {
	if err := service.Emit(ctx, "user.created", UserCreatedEvent{ID: "42", Name: "Typed Example"}, nil); err != nil {
		log.Printf("emit user.created: %v", err)
	}

	if err := service.Emit(ctx, "user.renamed", UserRenamedEvent{ID: "42", NewName: "Typed Example v2"}, nil); err != nil {
		log.Printf("emit user.renamed: %v", err)
	}
}

func callExamples(ctx context.Context, service rpc.NatsRPC) {
	var getResp GetUserResponse
	if err := service.CallRPC(ctx, "user.get", GetUserRequest{ID: "42"}, &getResp); err != nil {
		log.Printf("rpc call user.get: %v", err)
	} else {
		log.Printf("user.get response: %#v", getResp)
	}

	var updateResp UpdateUserResponse
	if err := service.CallRPC(ctx, "user.update", UpdateUserRequest{ID: "42", Name: "Typed Example v2"}, &updateResp); err != nil {
		log.Printf("rpc call user.update: %v", err)
	} else {
		log.Printf("user.update response: %#v", updateResp)
	}
}
