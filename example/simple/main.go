package main

import (
	"context"

	"github.com/nats-io/nats.go"
)

type RpcLib interface {
	RegisterRpc(method string, handler func())
}

func marshallerTest() {

}

func main() {
	// Connect to nats cluster
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	nc.ChanSubscribe()
}

type MyRequst struct {
	UserID string
}

type MyResponse struct {
	Name string
}

type rpcResponse struct {
	Data any
	Err  string
}

type RPCRequestHeaders interface {
	Get(k string) string
}
type RPCResponseHeaders interface {
	RPCRequestHeaders
	Set(k string, v string)
	Del(k string)
}

type RPCContext interface {
	context.Context
	Request(data any) error

	Ok(data any) error

	RequestHeaders() RPCRequestHeaders
	Headers() RPCResponseHeaders
}

func srakaHandler(c RPCContext) error {
	var msg MyRequst
	err := c.Request(&msg)
	if err != nil {
		return err
	}

	data, err := &MyResponse{}, nil
	if err != nil {
		return err
	}
	srakaHeader := c.RequestHeaders().Get("sraka")
	c.Headers().Set("X-User-ID", srakaHeader)

	return c.Ok(data)
}
