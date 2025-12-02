package natsevents

import (
	"github.com/leinodev/deez-nats/internal/router"
)

type eventRouterImpl[TMessage any, TAckOptFunc any, THandlerOption any, TMiddlewareFunc any] struct {
	base *router.Base[HandlerFunc[TMessage, TAckOptFunc], TMiddlewareFunc, THandlerOption]
}

func newEventRouter[TMessage any, TAckOptFunc any, THandlerOption any, TMiddlewareFunc any](group string, defaultOpts THandlerOption) *eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc] {
	return &eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]{
		base: router.NewTreeRouter[HandlerFunc[TMessage, TAckOptFunc], TMiddlewareFunc](group, defaultOpts),
	}
}

func (r *eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]) dfs() []router.Record[HandlerFunc[TMessage, TAckOptFunc], TMiddlewareFunc, THandlerOption] {
	return r.base.DFS()
}
func (r *eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]) Use(middlewares ...TMiddlewareFunc) {
	r.base.Use(middlewares...)
}
func (r *eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]) AddEventHandler(subject string, handler HandlerFunc[TMessage, TAckOptFunc], opts ...func(*THandlerOption)) {
	if subject == "" {
		panic("empty event subject name")
	}

	defaultOpts := r.base.DefaultOptions()
	options := defaultOpts

	// Apply functional options
	for _, opt := range opts {
		opt(&options)
	}

	r.base.Add(subject, handler, options)
}
func (r *eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]) Group(group string) EventRouter[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc] {
	child := r.base.Child(group)
	return &eventRouterImpl[TMessage, TAckOptFunc, THandlerOption, TMiddlewareFunc]{base: child}
}
