package main

import (
	"context"
	"fmt"
	"nrpc/protocol"
	"time"

	"github.com/leinodev/deez-nats/natsrpc"
	"github.com/nats-io/nats.go"
)

func main() {
	nats, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	nrpc := natsrpc.New(nats, natsrpc.WithBaseRoute("myservice"))

	server(nrpc)

	client(nrpc)
}

func server(nrpc natsrpc.NatsRPC) {
	// Register some method
	nrpc.AddRPCHandler("test.somemethod", func(c natsrpc.RPCContext) error {
		var r protocol.HelloRequest
		if err := c.Request(&r); err != nil {
			return err
		}

		return c.Ok(&protocol.HelloResponse{
			SentTime: time.Now().UnixNano(),
			Response: &protocol.SomeStruct{
				AnotherNumber: r.Request.AnotherNumber + 123,
				AnotherText:   r.Request.AnotherText + " from server",
			},
		})
	})
	err := nrpc.StartWithContext(context.Background())
	if err != nil {
		panic(err)
	}
}

func client(nrpc natsrpc.NatsRPC) {
	var resp protocol.HelloResponse

	roundtripTimeNsAvg := float64(0)
	server2clintNsAvg := float64(0)

	for i := range 100 {
		sentTime := time.Now().UnixNano()
		err := nrpc.CallRPC(context.Background(), "myservice.test.somemethod", &protocol.HelloRequest{
			SentTime: sentTime,
			Request: &protocol.SomeStruct{
				AnotherNumber: 321,
				AnotherText:   "Hello",
			},
		}, &resp)
		recvTime := time.Now().UnixNano()
		if err != nil {
			panic(err)
		}

		roundtripTimeNs := float64(recvTime - sentTime)
		server2ClientTimeNs := float64(recvTime - resp.SentTime)

		roundtripTimeNsAvg = (roundtripTimeNsAvg*float64(i) + roundtripTimeNs) / float64(i+1)
		server2clintNsAvg = (server2clintNsAvg*float64(i) + server2ClientTimeNs) / float64(i+1)
	}
	fmt.Println("------------------------------------")
	fmt.Printf("Avg roundtrip time: %v\n", time.Duration(roundtripTimeNsAvg*float64(time.Nanosecond)))
	fmt.Printf("Avg respond time: %v\n", time.Duration(server2clintNsAvg*float64(time.Nanosecond)))
	fmt.Println("------------------------------------")
}
