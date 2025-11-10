package natsrpcgo

import "fmt"

type RPCRouter interface {
	Use(middlewares ...RpcMiddlewareFunc)
	AddRPC(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc)
	Group(group string) RPCRouter

	dfs() []rpcInfo
}

type rpcInfo struct {
	method      string
	handler     RpcHandleFunc
	options     HandlerOptions
	middlewares []RpcMiddlewareFunc
}

type rpcRouterImpl struct {
	group string

	middlewares []RpcMiddlewareFunc
	rpcs        []rpcInfo
	daughters   []RPCRouter

	defaultOpts HandlerOptions
}

func newRouter(groupName string, defaultOpts HandlerOptions) RPCRouter {
	return &rpcRouterImpl{
		group:       groupName,
		defaultOpts: defaultOpts,
		middlewares: []RpcMiddlewareFunc{},
		rpcs:        []rpcInfo{},
		daughters:   []RPCRouter{},
	}
}

func (r *rpcRouterImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *rpcRouterImpl) AddRPC(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc) {
	options := r.defaultOpts
	if opts != nil {
		options = *opts
	}

	if len(method) == 0 {
		panic("empty rpc method name")
	}
	r.rpcs = append(r.rpcs, rpcInfo{
		method:      method,
		handler:     handler,
		options:     options,
		middlewares: middlewares,
	})
}

func (r *rpcRouterImpl) Group(group string) RPCRouter {
	daughterRouter := newRouter(group, r.defaultOpts)
	r.daughters = append(r.daughters, daughterRouter)
	return daughterRouter
}

func (r *rpcRouterImpl) dfs() []rpcInfo {
	allRoutes := make([]rpcInfo, 0)

	for _, router := range r.daughters {
		dRoutes := router.dfs()
		for _, route := range dRoutes {
			mRoute := rpcInfo{
				method:      route.method,
				handler:     route.handler,
				options:     route.options,
				middlewares: make([]RpcMiddlewareFunc, 0),
			}
			// collect all middlewares
			mRoute.middlewares = append(mRoute.middlewares, route.middlewares...)
			mRoute.middlewares = append(mRoute.middlewares, r.middlewares...)
			allRoutes = append(allRoutes, mRoute)
		}
	}
	allRoutes = append(allRoutes, r.rpcs...)

	for i, route := range allRoutes {
		allRoutes[i].method = r.joinPath(route.method)
	}
	return allRoutes
}

func (r *rpcRouterImpl) joinPath(method string) string {
	if len(r.group) != 0 {
		return fmt.Sprintf("%s.%s", r.group, method)
	}
	return method
}
