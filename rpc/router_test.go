package rpc

import (
	"fmt"
	"testing"

	"github.com/leinodev/deez-nats/marshaller"
)

func TestRouterDFS(t *testing.T) {
	r := newRouter("appname", HandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	})

	r.AddRPCHandler("health", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	r.AddRPCHandler("metrics", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	r.AddRPCHandler("internal.some.test", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	{
		entityGroup := r.Group("entity")

		entityGroup.AddRPCHandler("get", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPCHandler("create", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPCHandler("delete", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPCHandler("any", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})

		entityListGroup := entityGroup.Group("list")

		entityListGroup.AddRPCHandler("byID", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityListGroup.AddRPCHandler("byName", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
	}

	routes := r.dfs()
	fmt.Println(routes)
}
