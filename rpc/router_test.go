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
	})

	r.AddRPCHandler("metrics", func(c RPCContext) error {
		return nil
	})

	r.AddRPCHandler("internal.some.test", func(c RPCContext) error {
		return nil
	})

	{
		entityGroup := r.Group("entity")

		entityGroup.AddRPCHandler("get", func(c RPCContext) error {
			return nil
		})
		entityGroup.AddRPCHandler("create", func(c RPCContext) error {
			return nil
		})
		entityGroup.AddRPCHandler("delete", func(c RPCContext) error {
			return nil
		})
		entityGroup.AddRPCHandler("any", func(c RPCContext) error {
			return nil
		})

		entityListGroup := entityGroup.Group("list")

		entityListGroup.AddRPCHandler("byID", func(c RPCContext) error {
			return nil
		})
		entityListGroup.AddRPCHandler("byName", func(c RPCContext) error {
			return nil
		})
	}

	routes := r.dfs()
	fmt.Println(routes)
}
