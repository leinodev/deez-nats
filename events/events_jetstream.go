package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/internal/middleware"
	"github.com/leinodev/deez-nats/internal/router"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type jetStreamNatsEventsImpl struct {
	router  *eventRouterImpl[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]]
	js      jetstream.JetStream
	options JetStreamEventsOptions

	handlersWatch sync.WaitGroup
	subsTracker   *subscriptions.Tracker
}

func NewJetStreamEvents(js jetstream.JetStream, opts ...JetStreamEventsOptionFunc) JetStreamNatsEvents {
	options := JetStreamEventsOptions{
		DefaultEmitMarshaller:         marshaller.DefaultJsonMarshaller,
		DefaultEventHandlerMarshaller: marshaller.DefaultJsonMarshaller,
	}
	for _, opt := range opts {
		opt(&options)
	}

	handlerOptions := JetStreamEventHandlerOptions{
		Marshaller: options.DefaultEventHandlerMarshaller,
	}

	return &jetStreamNatsEventsImpl{
		js:          js,
		options:     options,
		router:      newEventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]]("", handlerOptions),
		subsTracker: subscriptions.NewTracker(),
	}
}

// Router inherited
func (e *jetStreamNatsEventsImpl) Use(middlewares ...MiddlewareFunc[jetstream.Msg, any]) {
	e.router.Use(middlewares...)
}
func (e *jetStreamNatsEventsImpl) AddEventHandler(subject string, handler HandlerFunc[jetstream.Msg, any], opts ...func(*JetStreamEventHandlerOptions)) {
	e.router.AddEventHandler(subject, handler, opts...)
}
func (e *jetStreamNatsEventsImpl) Group(group string) EventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]] {
	return e.router.Group(group)
}

// methods
func (e *jetStreamNatsEventsImpl) StartWithContext(ctx context.Context) error {
	var sub jetstream.Consumer
	var err error

	for _, route := range e.router.dfs() {
		handler := e.wrapHandler(ctx, route)

		// TODO: add options and maybe pull consumer
		consumerConfig := jetstream.ConsumerConfig{
			DeliverGroup: e.options.DeliverGroup,
		}

		sub, err = e.js.CreateOrUpdateConsumer(ctx, e.options.Stream, consumerConfig)
		if err != nil {
			e.Shutdown(ctx)

			return fmt.Errorf("failed to subscribe %s: %w", route.Name, err)
		}

		consumeCtx, err := sub.Consume(handler)
		if err != nil {
			e.Shutdown(ctx)

			return fmt.Errorf("failed to consume %s: %w", route.Name, err)
		}

		e.subsTracker.Track(subscriptions.NewJsSub(consumeCtx))
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()

	return nil
}
func (e *jetStreamNatsEventsImpl) Emit(ctx context.Context, subject string, payload any, opts ...func(*JetStreamEventEmitOptions)) error {
	if subject == "" {
		return ErrEmptySubject
	}

	emitOptions := JetStreamEventEmitOptions{
		Marshaller: e.options.DefaultEmitMarshaller,
	}
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
		Header:  mergeHeaders(e.options.DefaultEmitHeaders, emitOptions.Headers),
	}

	// TODO: pass options
	_, err = e.js.PublishMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}
func (e *jetStreamNatsEventsImpl) Shutdown(ctx context.Context) error {
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
func (e *jetStreamNatsEventsImpl) wrapHandler(
	ctx context.Context,
	route router.Record[HandlerFunc[jetstream.Msg, any], MiddlewareFunc[jetstream.Msg, any], JetStreamEventHandlerOptions],
) jetstream.MessageHandler {
	handler := middleware.Apply(route.Handler, route.Middlewares, true)
	handlerOptions := route.Options

	return func(msg jetstream.Msg) {
		e.handlersWatch.Add(1)
		defer e.handlersWatch.Done()

		eventCtx := newJetStreamContext(ctx, msg, handlerOptions.Marshaller)
		err := handler(eventCtx)

		if err != nil {
			eventCtx.Nak()
			return
		}

		eventCtx.Ack()
	}
}
