package natsrpcgo

type RPCRouter interface {
	Use(middlewares ...RpcMiddlewareFunc)
	AddRPCHandler(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc)
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
	base *routerBase[RpcHandleFunc, RpcMiddlewareFunc, HandlerOptions]
}

func newRouter(groupName string, defaultOpts HandlerOptions) RPCRouter {
	return &rpcRouterImpl{
		base: newRouterBase[RpcHandleFunc, RpcMiddlewareFunc, HandlerOptions](groupName, defaultOpts),
	}
}

func (r *rpcRouterImpl) Use(middlewares ...RpcMiddlewareFunc) {
	r.base.use(middlewares...)
}

func (r *rpcRouterImpl) AddRPCHandler(method string, handler RpcHandleFunc, opts *HandlerOptions, middlewares ...RpcMiddlewareFunc) {
	if len(method) == 0 {
		panic("empty rpc method name")
	}

	options := r.base.defaultOpts
	if opts != nil {
		options = *opts
	}

	r.base.add(method, handler, options, middlewares...)
}

func (r *rpcRouterImpl) Group(group string) RPCRouter {
	child := r.base.child(group)
	return &rpcRouterImpl{base: child}
}

func (r *rpcRouterImpl) dfs() []rpcInfo {
	records := r.base.dfs()
	routes := make([]rpcInfo, 0, len(records))
	for _, rec := range records {
		routes = append(routes, rpcInfo{
			method:      rec.name,
			handler:     rec.handler,
			options:     rec.options,
			middlewares: rec.middlewares,
		})
	}
	return routes
}
