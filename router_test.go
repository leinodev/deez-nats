package natsrpcgo

import (
	"fmt"
	"testing"

	"github.com/TexHik620953/natsrpc-go/marshaller"
)

func TestRouterDFS(t *testing.T) {
	r := newRouter("appname", HandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	})

	r.AddRPC("health", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	r.AddRPC("metrics", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	r.AddRPC("internal.some.test", func(c RPCContext) error {
		return nil
	}, &HandlerOptions{})

	{
		entityGroup := r.Group("entity")

		entityGroup.AddRPC("get", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPC("create", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPC("delete", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityGroup.AddRPC("any", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})

		entityListGroup := entityGroup.Group("list")

		entityListGroup.AddRPC("byID", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
		entityListGroup.AddRPC("byName", func(c RPCContext) error {
			return nil
		}, &HandlerOptions{})
	}

	routes := r.dfs()
	fmt.Println(routes)
}
