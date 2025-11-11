package natsrpcgo

type routeRecord[THandler any, TMiddleware any, TOptions any] struct {
	name        string
	handler     THandler
	options     TOptions
	middlewares []TMiddleware
}

type routerBase[THandler any, TMiddleware any, TOptions any] struct {
	group       string
	defaultOpts TOptions

	middlewares []TMiddleware
	routes      []routeRecord[THandler, TMiddleware, TOptions]
	childs      []*routerBase[THandler, TMiddleware, TOptions]
}

func newRouterBase[THandler any, TMiddleware any, TOptions any](group string, defaultOpts TOptions) *routerBase[THandler, TMiddleware, TOptions] {
	return &routerBase[THandler, TMiddleware, TOptions]{
		group:       group,
		defaultOpts: defaultOpts,
		middlewares: make([]TMiddleware, 0),
		routes:      make([]routeRecord[THandler, TMiddleware, TOptions], 0),
		childs:      make([]*routerBase[THandler, TMiddleware, TOptions], 0),
	}
}

func (r *routerBase[THandler, TMiddleware, TOptions]) use(middlewares ...TMiddleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *routerBase[THandler, TMiddleware, TOptions]) add(name string, handler THandler, opts TOptions, middlewares ...TMiddleware) {
	route := routeRecord[THandler, TMiddleware, TOptions]{
		name:        name,
		handler:     handler,
		options:     opts,
		middlewares: append([]TMiddleware(nil), middlewares...),
	}
	r.routes = append(r.routes, route)
}

func (r *routerBase[THandler, TMiddleware, TOptions]) child(name string) *routerBase[THandler, TMiddleware, TOptions] {
	child := newRouterBase[THandler, TMiddleware, TOptions](name, r.defaultOpts)
	r.childs = append(r.childs, child)
	return child
}

func (r *routerBase[THandler, TMiddleware, TOptions]) dfs() []routeRecord[THandler, TMiddleware, TOptions] {
	allRoutes := make([]routeRecord[THandler, TMiddleware, TOptions], 0)

	for _, child := range r.childs {
		childRoutes := child.dfs()
		for _, route := range childRoutes {
			merged := routeRecord[THandler, TMiddleware, TOptions]{
				name:        r.join(route.name),
				handler:     route.handler,
				options:     route.options,
				middlewares: append([]TMiddleware(nil), route.middlewares...),
			}
			merged.middlewares = append(merged.middlewares, r.middlewares...)
			allRoutes = append(allRoutes, merged)
		}
	}

	for _, route := range r.routes {
		merged := routeRecord[THandler, TMiddleware, TOptions]{
			name:        r.join(route.name),
			handler:     route.handler,
			options:     route.options,
			middlewares: append([]TMiddleware(nil), route.middlewares...),
		}
		merged.middlewares = append(merged.middlewares, r.middlewares...)
		allRoutes = append(allRoutes, merged)
	}

	return allRoutes
}

func (r *routerBase[THandler, TMiddleware, TOptions]) join(name string) string {
	if r.group == "" {
		return name
	}
	if name == "" {
		return r.group
	}
	return r.group + "." + name
}
