package natsrpc

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type RouteType string

const (
	RouteTypeUnary           RouteType = "urany"
	RouteTypeBidirectional   RouteType = "bidirectional"
	RouteTypeServerStreaming RouteType = "server_stream"
	RouteTypeClientStreaming RouteType = "client_stream"
)

type routeInfo struct {
	Type RouteType

	method      string
	handler     RpcUnaryHandleFunc
	options     HandlerOptions
	middlewares []RpcUnaryMiddlewareFunc

	sub *nats.Subscription
}

type RPCRouter interface {
	Use(middlewares ...RpcUnaryMiddlewareFunc)
	AddUnaryHandler(method string, handler RpcUnaryHandleFunc, opts ...HandlerOption)
	Group(group string) RPCRouter

	dfs() []routeInfo
}

type rpcRouterImpl struct {
	group       string
	defaultOpts HandlerOptions

	middlewares []RpcUnaryMiddlewareFunc
	routes      []routeInfo
	childs      []RPCRouter
}

func newRouter(groupName string, defaultOpts HandlerOptions) RPCRouter {
	return &rpcRouterImpl{
		group:       groupName,
		defaultOpts: defaultOpts,
	}
}

func (r *rpcRouterImpl) Use(middlewares ...RpcUnaryMiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *rpcRouterImpl) AddUnaryHandler(method string, handler RpcUnaryHandleFunc, opts ...HandlerOption) {
	if len(method) == 0 {
		panic("empty rpc method name")
	}

	options := r.defaultOpts

	// Apply functional options
	for _, opt := range opts {
		opt(&options)
	}

	r.routes = append(r.routes, routeInfo{
		method:  method,
		handler: handler,
		options: options,
	})
}

func (r *rpcRouterImpl) Group(group string) RPCRouter {
	child := newRouter(group, r.defaultOpts)
	r.childs = append(r.childs, child)
	return child
}

func (r *rpcRouterImpl) dfs() []routeInfo {
	allRoutes := make([]routeInfo, 0)

	for _, child := range r.childs {
		for _, route := range child.dfs() {
			merged := routeInfo{
				method:      r.join(route.method),
				handler:     route.handler,
				options:     route.options,
				middlewares: append(route.middlewares, r.middlewares...),
			}
			allRoutes = append(allRoutes, merged)
		}
	}

	for _, route := range r.routes {
		merged := routeInfo{
			method:      r.join(route.method),
			handler:     route.handler,
			options:     route.options,
			middlewares: append(route.middlewares, r.middlewares...),
		}
		allRoutes = append(allRoutes, merged)
	}

	return allRoutes
}

func (r *rpcRouterImpl) join(name string) string {
	if r.group == "" {
		return name
	}
	if name == "" {
		return r.group
	}
	return fmt.Sprintf("%s.%s", r.group, name)
}
