package events

import (
	"github.com/leinodev/deez-nats/internal/router"
)

type eventInfo struct {
	subject     string
	handler     EventHandleFunc
	options     EventHandlerOptions
	middlewares []EventMiddlewareFunc
}

type eventRouterImpl struct {
	base *router.Base[EventHandleFunc, EventMiddlewareFunc, EventHandlerOptions]
}

func newEventRouter(groupName string, defaultOpts EventHandlerOptions) EventRouter {
	return &eventRouterImpl{
		base: router.NewBase[EventHandleFunc, EventMiddlewareFunc](groupName, defaultOpts),
	}
}

func (r *eventRouterImpl) Use(middlewares ...EventMiddlewareFunc) {
	r.base.Use(middlewares...)
}

func (r *eventRouterImpl) AddEventHandler(subject string, handler EventHandleFunc, opts ...EventHandlerOption) {
	r.AddEventHandlerWithMiddlewares(subject, handler, nil, opts...)
}

func (r *eventRouterImpl) AddEventHandlerWithMiddlewares(subject string, handler EventHandleFunc, middlewares []EventMiddlewareFunc, opts ...EventHandlerOption) {
	if subject == "" {
		panic("empty event subject name")
	}
	options := NewEventHandlerOptions(opts...)

	r.base.Add(subject, handler, options, middlewares...)
}

func (r *eventRouterImpl) Group(group string) EventRouter {
	child := r.base.Child(group)
	return &eventRouterImpl{base: child}
}

func (r *eventRouterImpl) dfs() []eventInfo {
	records := r.base.DFS()
	routes := make([]eventInfo, 0, len(records))
	for _, rec := range records {
		routes = append(routes, eventInfo{
			subject:     rec.Name,
			handler:     rec.Handler,
			options:     rec.Options,
			middlewares: rec.Middlewares,
		})
	}
	return routes
}
