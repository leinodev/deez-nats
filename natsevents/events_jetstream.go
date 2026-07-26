package natsevents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/internal/router"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/internal/utils"
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

type jetStreamRoute = router.Record[
	HandlerFunc[jetstream.Msg, any],
	MiddlewareFunc[jetstream.Msg, any],
	JetStreamEventHandlerOptions,
]

func NewJetStream(js jetstream.JetStream, opts ...JetStreamEventsOptionFunc) JetStreamNatsEvents {
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
	if strings.TrimSpace(e.options.Stream) == "" {
		return fmt.Errorf("jetstream stream is required (use WithJetStreamStream)")
	}

	routes := e.router.dfs()
	for _, route := range routes {
		if err := e.startRoute(ctx, route, len(routes)); err != nil {
			_ = e.Shutdown(ctx)
			return err
		}
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()

	return nil
}

func (e *jetStreamNatsEventsImpl) startRoute(
	ctx context.Context,
	route jetStreamRoute,
	routeCount int,
) error {
	handler := e.wrapHandler(ctx, route)
	config := e.consumerConfig(route, routeCount)
	consumer, err := e.js.CreateOrUpdateConsumer(ctx, e.options.Stream, config)
	if err != nil {
		return fmt.Errorf("failed to subscribe %s: %w", route.Name, err)
	}
	consumeCtx, err := consumer.Consume(handler)
	if err != nil {
		return fmt.Errorf("failed to consume %s: %w", route.Name, err)
	}
	e.subsTracker.Track(subscriptions.NewJsSub(consumeCtx))
	return nil
}

func (e *jetStreamNatsEventsImpl) consumerConfig(
	route jetStreamRoute,
	routeCount int,
) jetstream.ConsumerConfig {
	filterSubjects := []string{route.Name}
	if len(route.Options.FilterSubjects) > 0 {
		filterSubjects = append([]string(nil), route.Options.FilterSubjects...)
	}
	durable := strings.TrimSpace(route.Options.ConsumerDurable)
	if durable == "" {
		durable = jetStreamPushDurable(e.options.ConsumerDurable, route.Name, routeCount)
	} else {
		durable = sanitizeJetStreamDurable(durable)
	}
	config := jetstream.ConsumerConfig{
		DeliverGroup: e.options.DeliverGroup, FilterSubjects: filterSubjects,
		Durable: durable, AckPolicy: e.options.ConsumerAckPolicy,
	}
	if config.AckPolicy == 0 {
		config.AckPolicy = jetstream.AckExplicitPolicy
	}
	if e.options.ConsumerAckWait > 0 {
		config.AckWait = e.options.ConsumerAckWait
	}
	if e.options.ConsumerMaxDeliver > 0 {
		config.MaxDeliver = e.options.ConsumerMaxDeliver
	}
	return config
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
	handler := utils.ApplyMiddlewares(route.Handler, route.Middlewares, true)

	return func(msg jetstream.Msg) {
		e.handlersWatch.Add(1)
		defer e.handlersWatch.Done()

		eventCtx := newJetStreamContext(ctx, msg, route.Options.Marshaller)
		err := handler(eventCtx)

		if err != nil {
			_ = eventCtx.Nak()
			return
		}

		_ = eventCtx.Ack()
	}
}
