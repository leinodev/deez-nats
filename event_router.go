package natsrpcgo

import "github.com/TexHik620953/natsrpc-go/marshaller"

type EventHandleFunc func(EventContext) error
type EventMiddlewareFunc func(next EventHandleFunc) EventHandleFunc

type EventRouter interface {
	Use(middlewares ...EventMiddlewareFunc)
	AddEventHandler(subject string, handler EventHandleFunc, opts *EventHandlerOptions, middlewares ...EventMiddlewareFunc)
	Group(group string) EventRouter

	dfs() []eventInfo
}

type eventInfo struct {
	subject     string
	handler     EventHandleFunc
	options     EventHandlerOptions
	middlewares []EventMiddlewareFunc
}

type eventRouterImpl struct {
	base *routerBase[EventHandleFunc, EventMiddlewareFunc, EventHandlerOptions]
}

func newEventRouter(groupName string, defaultOpts EventHandlerOptions) EventRouter {
	if defaultOpts.Marshaller == nil {
		defaultOpts.Marshaller = marshaller.DefaultJsonMarshaller
	}

	return &eventRouterImpl{
		base: newRouterBase[EventHandleFunc, EventMiddlewareFunc, EventHandlerOptions](groupName, defaultOpts),
	}
}

func (r *eventRouterImpl) Use(middlewares ...EventMiddlewareFunc) {
	r.base.use(middlewares...)
}

func (r *eventRouterImpl) AddEventHandler(subject string, handler EventHandleFunc, opts *EventHandlerOptions, middlewares ...EventMiddlewareFunc) {
	if subject == "" {
		panic("empty event subject name")
	}

	options := r.base.defaultOpts
	if options.Marshaller == nil {
		options.Marshaller = marshaller.DefaultJsonMarshaller
	}

	if opts != nil {
		options = *opts
		if options.Marshaller == nil {
			options.Marshaller = r.base.defaultOpts.Marshaller
		}
		if !opts.JetStream.Enabled {
			options.JetStream = JetStreamEventOptions{}
		}
	}

	if options.Marshaller == nil {
		options.Marshaller = marshaller.DefaultJsonMarshaller
	}

	r.base.add(subject, handler, options, middlewares...)
}

func (r *eventRouterImpl) Group(group string) EventRouter {
	child := r.base.child(group)
	return &eventRouterImpl{base: child}
}

func (r *eventRouterImpl) dfs() []eventInfo {
	records := r.base.dfs()
	routes := make([]eventInfo, 0, len(records))
	for _, rec := range records {
		routes = append(routes, eventInfo{
			subject:     rec.name,
			handler:     rec.handler,
			options:     rec.options,
			middlewares: rec.middlewares,
		})
	}
	return routes
}
