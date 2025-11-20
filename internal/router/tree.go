package router

type Record[THandler any, TMiddleware any, TOptions any] struct {
	Name        string
	Handler     THandler
	Options     TOptions
	Middlewares []TMiddleware
}

type Base[THandler any, TMiddleware any, TOptions any] struct {
	group       string
	defaultOpts TOptions

	middlewares []TMiddleware
	routes      []Record[THandler, TMiddleware, TOptions]
	childs      []*Base[THandler, TMiddleware, TOptions]
}

func NewTreeRouter[THandler any, TMiddleware any, TOptions any](group string, defaultOpts TOptions) *Base[THandler, TMiddleware, TOptions] {
	return &Base[THandler, TMiddleware, TOptions]{
		group:       group,
		defaultOpts: defaultOpts,
		middlewares: make([]TMiddleware, 0),
		routes:      make([]Record[THandler, TMiddleware, TOptions], 0),
		childs:      make([]*Base[THandler, TMiddleware, TOptions], 0),
	}
}

func (r *Base[THandler, TMiddleware, TOptions]) Use(middlewares ...TMiddleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Base[THandler, TMiddleware, TOptions]) Add(name string, handler THandler, opts TOptions, middlewares ...TMiddleware) {
	route := Record[THandler, TMiddleware, TOptions]{
		Name:        name,
		Handler:     handler,
		Options:     opts,
		Middlewares: append([]TMiddleware(nil), middlewares...),
	}
	r.routes = append(r.routes, route)
}

func (r *Base[THandler, TMiddleware, TOptions]) Child(name string) *Base[THandler, TMiddleware, TOptions] {
	child := NewTreeRouter[THandler, TMiddleware](name, r.defaultOpts)
	r.childs = append(r.childs, child)
	return child
}

func (r *Base[THandler, TMiddleware, TOptions]) DefaultOptions() TOptions {
	return r.defaultOpts
}

func (r *Base[THandler, TMiddleware, TOptions]) DFS() []Record[THandler, TMiddleware, TOptions] {
	allRoutes := make([]Record[THandler, TMiddleware, TOptions], 0)

	for _, child := range r.childs {
		childRoutes := child.DFS()
		for _, route := range childRoutes {
			merged := Record[THandler, TMiddleware, TOptions]{
				Name:        r.join(route.Name),
				Handler:     route.Handler,
				Options:     route.Options,
				Middlewares: append([]TMiddleware(nil), route.Middlewares...),
			}
			merged.Middlewares = append(merged.Middlewares, r.middlewares...)
			allRoutes = append(allRoutes, merged)
		}
	}

	for _, route := range r.routes {
		merged := Record[THandler, TMiddleware, TOptions]{
			Name:        r.join(route.Name),
			Handler:     route.Handler,
			Options:     route.Options,
			Middlewares: append([]TMiddleware(nil), route.Middlewares...),
		}
		merged.Middlewares = append(merged.Middlewares, r.middlewares...)
		allRoutes = append(allRoutes, merged)
	}

	return allRoutes
}

func (r *Base[THandler, TMiddleware, TOptions]) join(name string) string {
	if r.group == "" {
		return name
	}
	if name == "" {
		return r.group
	}
	return r.group + "." + name
}
