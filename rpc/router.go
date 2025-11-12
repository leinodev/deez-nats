package rpc

import "github.com/leinodev/deez-nats/internal/router"

type rpcInfo struct {
	method      string
	handler     RpcHandleFunc
	options     HandlerOptions
	middlewares []RpcMiddlewareFunc
}

type rpcRouterImpl struct {
	base *router.Base[RpcHandleFunc, RpcMiddlewareFunc, HandlerOptions]
}

func newRouter(groupName string, defaultOpts HandlerOptions) RPCRouter {
	return &rpcRouterImpl{
		base: router.NewBase[RpcHandleFunc, RpcMiddlewareFunc](groupName, defaultOpts),
	}
}

func (r *rpcRouterImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.base.Use(middlewares...)
}

func (r *rpcRouterImpl) AddRPCHandler(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc) {
	if len(method) == 0 {
		panic("empty rpc method name")
	}
	defaultOpts := r.base.DefaultOptions()
	options := defaultOpts
	if opts != nil {
		options = *opts

		if options.Marshaller == nil {
			options.Marshaller = defaultOpts.Marshaller
		}
	}

	r.base.Add(method, handler, options, middlewares...)
}

func (r *rpcRouterImpl) Group(group string) RPCRouter {
	child := r.base.Child(group)
	return &rpcRouterImpl{base: child}
}

func (r *rpcRouterImpl) dfs() []rpcInfo {
	records := r.base.DFS()
	routes := make([]rpcInfo, 0, len(records))
	for _, rec := range records {
		routes = append(routes, rpcInfo{
			method:      rec.Name,
			handler:     rec.Handler,
			options:     rec.Options,
			middlewares: rec.Middlewares,
		})
	}
	return routes
}
