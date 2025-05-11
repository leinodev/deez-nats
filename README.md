# natsrpc-go

[![Go Reference](https://pkg.go.dev/badge/github.com/TexHik620953/natsrpc-go.svg)](https://pkg.go.dev/github.com/TexHik620953/natsrpc-go)

natsrpc is a lightweight wrapper for NATS that provides a convenient RPC interface for Go applications. The library makes it easy to register and call remote methods through NATS.

## Features

- Simple creation of RPC services on top of NATS
- JSON serialization support
- Context for request processing
- Flexible subject naming configuration
- Automatic error handling

## Installation

```bash
go get github.com/TexHik620953/natsrpc-go
```

## Quickstart example

```go
package main

import (
	"context"
	"fmt"

	"github.com/TexHik620953/natsrpc-go"
	"github.com/nats-io/nats.go"
)

// Define request and response structures
type ExampleRequest struct {
	Text string
}
type ExampleResponse struct {
	ModifiedText string
	SymbolsCount int
}

// RPC method handler 1
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

// RPC method handler 2
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
	// Connect to NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}

	// Create natsrpc instance with base name "testapp"
	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))

	// Register handlers
	nrpc.AddRPC("f1", greeterfunc1)
	nrpc.AddRPC("f2", greeterfunc2)

	// Start subscriptions
	err = nrpc.StartWithContext(context.Background())
	if err != nil {
		panic(err)
	}

	// Call first handler
	{
		var resp ExampleResponse
		err = nrpc.CallRPC("testapp.f1", &ExampleRequest{Text: "natsrpc1"}, &resp)
		if err != nil {
			panic(err)
		}

		fmt.Println(resp.ModifiedText, resp.SymbolsCount) // Output: Hello, natsrpc1 8
	}

	// Call second handler
	{
		var resp ExampleResponse
		err = nrpc.CallRPC("testapp.f2", &ExampleRequest{Text: "natsrpc2"}, &resp)
		if err != nil {
			panic(err)
		}

		fmt.Println(resp.ModifiedText, resp.SymbolsCount) // Output: Bye, natsrpc2 -8
	}
}
```