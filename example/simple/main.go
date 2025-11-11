package main

import (
	"context"
	"errors"
	"log"
	"simple/protocoljson"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	// Connect to nats cluster
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	events := NewEventRouter(nc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := events.StartWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("events stopped with error: %v", err)
		}
	}()

	// Emit example events
	err = events.Publish(ctx, "entity.created", protocoljson.EntityEvent{ID: "42", Name: "example"}, nil)
	if err != nil {
		log.Printf("publish entity.created: %v", err)
	}

	err = events.Publish(ctx, "entity.updated", protocoljson.EntityEvent{ID: "42", Name: "example-renamed"}, nil)
	if err != nil {
		log.Printf("publish entity.updated: %v", err)
	}

	time.Sleep(2 * time.Second)
}
