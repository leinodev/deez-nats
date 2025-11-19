package newevents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/internal/router"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type coreNatsEventsImpl struct {
	router     *eventRouterImpl[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]]
	connection *nats.Conn
	options    CoreEventsOptions

	handlersWatch sync.WaitGroup
	subsTracker   *subscriptions.Tracker
}

func NewCoreEvents(nc *nats.Conn, opts ...CoreEventsOptionFunc) CoreNatsEvents {
	// TODO: default options
	options := CoreEventsOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	// TODO: default handler options
	handlerOptions := CoreEventHandlerOptions{}

	return &coreNatsEventsImpl{
		connection:  nc,
		router:      newEventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]]("", handlerOptions),
		options:     options,
		subsTracker: subscriptions.NewTracker(),
	}
}

// Router inherited
func (e *coreNatsEventsImpl) Use(middlewares ...MiddlewareFunc[*nats.Msg, nats.AckOpt]) {
	e.router.Use(middlewares...)
}
func (e *coreNatsEventsImpl) AddEventHandler(subject string, handler HandlerFunc[*nats.Msg, nats.AckOpt], opts ...func(*CoreEventHandlerOptions)) {
	e.router.AddEventHandler(subject, handler, opts...)
}
func (e *coreNatsEventsImpl) Group(group string) EventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]] {
	return e.router.Group(group)
}

// methods
func (e *coreNatsEventsImpl) StartWithContext(ctx context.Context) error {
	var sub *nats.Subscription
	var err error

	for _, route := range e.router.dfs() {
		handler := e.wrapHandler(ctx, route)

		if e.options.QueueGroup != "" {
			sub, err = e.connection.QueueSubscribe(route.Name, e.options.QueueGroup, handler)
		} else {
			sub, err = e.connection.Subscribe(route.Name, handler)
		}

		if err != nil {
			e.Shutdown(ctx)

			return fmt.Errorf("failed to subscribe %s: %w", route.Name, err)
		}

		e.subsTracker.Track(sub)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()

	return nil
}
func (e *coreNatsEventsImpl) Emit(ctx context.Context, subject string, payload any, opts ...func(*CoreEventEmitOptions)) error {
	if subject == "" {
		return ErrEmptySubject
	}

	// TODO: default options
	emitOptions := CoreEventEmitOptions{}
	for _, opt := range opts {
		opt(&emitOptions)
	}

	payloadBytes, err := emitOptions.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: payload,
	})
	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    payloadBytes,
		Header:  emitOptions.Headers,
	}

	return e.connection.PublishMsg(msg)
}
func (e *coreNatsEventsImpl) Shutdown(ctx context.Context) error {
	e.subsTracker.Drain()

	finished := make(chan struct{})
	go func() {
		e.handlersWatch.Wait()
		finished <- struct{}{}
		close(finished)
	}()

	select {
	case <-finished:
		break
	case <-ctx.Done():
		return fmt.Errorf("failed to wait for handlers finish: %w", context.DeadlineExceeded)
	}

	e.subsTracker.Unsubscribe() // Unsubscribe from all routes
	return nil
}

// internal methods
func (e *coreNatsEventsImpl) wrapHandler(ctx context.Context, route router.Record[HandlerFunc[*nats.Msg, nats.AckOpt], MiddlewareFunc[*nats.Msg, nats.AckOpt], CoreEventHandlerOptions]) nats.MsgHandler {
	return func(msg *nats.Msg) {
		e.handlersWatch.Add(1)
		defer e.handlersWatch.Done()

		eventCtx := newCoreContext(ctx, msg, route.Options.Marshaller)
		err := route.Handler(eventCtx)

		if err != nil {
			eventCtx.Nak()
			return
		}

		eventCtx.Ack()
	}
}
