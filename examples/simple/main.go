package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"simple/protocoljson"
	"syscall"
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
	if err := rpcService.CallRPC(ctx, "test", protocoljson.MyRequst{UserID: "42"}, &response, rpc.CallOptions{}); err != nil {
		log.Printf("rpc call test: %v", err)
	}

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
