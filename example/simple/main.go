package main

import (
	"context"
	"fmt"

	"github.com/TexHik620953/natsrpc-go"
	"github.com/nats-io/nats.go"
)

type ExampleRequest struct {
	Text string
}
type ExampleResponse struct {
	ModifiedText string
	SymbolsCount int
}

func greeterfunc1(c natsrpc.NatsRPCContext) error {
	var request ExampleRequest
	err := c.RequestJSON(&request)
	if err != nil {
		return err
	}

	return c.Ok(&ExampleResponse{
		ModifiedText: "Hello, " + request.Text,
		SymbolsCount: len(request.Text),
	})
}
func greeterfunc2(c natsrpc.NatsRPCContext) error {
	var request ExampleRequest
	err := c.RequestJSON(&request)
	if err != nil {
		return err
	}

	return c.Ok(&ExampleResponse{
		ModifiedText: "Bye, " + request.Text,
		SymbolsCount: -len(request.Text),
	})
}

func main() {
	// Connect to nats cluster
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}

	// Create natsrpc wrapper with defined base name
	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))

	// Register handlers
	nrpc.AddRPC("f1", greeterfunc1)
	nrpc.AddRPC("f2", greeterfunc2)

	// Start subscriptions
	err = nrpc.StartWithContext(context.Background())
	if err != nil {
		panic(err)
	}
	// First handler
	{
		var resp ExampleResponse
		err = nrpc.CallRPC("testapp.f1", &ExampleRequest{Text: "natsrpc1"}, &resp)
		if err != nil {
			panic(err)
		}

		fmt.Println(resp.ModifiedText, resp.SymbolsCount)
	}
	// Second handler
	{
		var resp ExampleResponse
		err = nrpc.CallRPC("testapp.f2", &ExampleRequest{Text: "natsrpc2"}, &resp)
		if err != nil {
			panic(err)
		}

		fmt.Println(resp.ModifiedText, resp.SymbolsCount)
	}
}
