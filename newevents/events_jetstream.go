package newevents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/internal/provider"
	"github.com/leinodev/deez-nats/internal/router"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type jsSubImpl struct {
	ctx jetstream.ConsumeContext
}

func newJsSub(ctx jetstream.ConsumeContext) provider.TransportSubscription {
	return &jsSubImpl{
		ctx: ctx,
	}
}

func (s *jsSubImpl) Drain() error {
	s.ctx.Drain()
	return nil
}

func (s *jsSubImpl) Unsubscribe() error {
	s.ctx.Stop()
	return nil
}

type jetStreamNatsEventsImpl struct {
	router  *eventRouterImpl[jetstream.Msg, nats.AckOpt, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, nats.AckOpt]]
	js      jetstream.JetStream
	options JetStreamEventsOptions

	handlersWatch sync.WaitGroup
	subsTracker   *subscriptions.Tracker
}

func NewJetStreamEvents(js jetstream.JetStream, opts ...JetStreamEventsOptionFunc) JetStreamNatsEvents {
	// TODO: default options
	options := JetStreamEventsOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	// TODO: default handler options
	handlerOptions := JetStreamEventHandlerOptions{}

	return &jetStreamNatsEventsImpl{
		js:      js,
		options: options,
		router:  newEventRouter[jetstream.Msg, nats.AckOpt, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, nats.AckOpt]]("", handlerOptions),
	}
}

// Router inherited
func (e *jetStreamNatsEventsImpl) Use(middlewares ...MiddlewareFunc[jetstream.Msg, nats.AckOpt]) {
	e.router.Use(middlewares...)
}
func (e *jetStreamNatsEventsImpl) AddEventHandler(subject string, handler HandlerFunc[jetstream.Msg, nats.AckOpt], opts ...func(JetStreamEventHandlerOptions)) {
	e.router.AddEventHandler(subject, handler, opts...)
}
func (e *jetStreamNatsEventsImpl) Group(group string) EventRouter[jetstream.Msg, nats.AckOpt, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, nats.AckOpt]] {
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

		e.subsTracker.Track(newJsSub(consumeCtx))
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

	// TODO: default options
	emitOptions := JetStreamEventEmitOptions{}
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
	route router.Record[HandlerFunc[jetstream.Msg, nats.AckOpt], MiddlewareFunc[jetstream.Msg, nats.AckOpt], JetStreamEventHandlerOptions],
) jetstream.MessageHandler {
	return func(msg jetstream.Msg) {
		e.handlersWatch.Add(1)
		defer e.handlersWatch.Done()

		// TODO: handle logic

		// eventCtx := newCoreEventContext(ctx, msg, route.options.Marshaller)
		// return route.handler(eventCtx)
	}
}
