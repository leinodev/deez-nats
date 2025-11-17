package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/internal/graceful"
	"github.com/leinodev/deez-nats/internal/lifecycle"
	"github.com/leinodev/deez-nats/internal/middleware"
	"github.com/leinodev/deez-nats/internal/provider"
	"github.com/leinodev/deez-nats/internal/subscriptions"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

var (
	ErrJetStreamPullRequiresDurable = errors.New("jetstream pull consumer requires durable name")
	ErrEmptySubject                 = errors.New("empty subject")
)

type natsEventsImpl struct {
	nc         *nats.Conn
	options    EventsOptions
	rootRouter EventRouter

	lifecycleMgr *lifecycle.Manager
	subsTracker  *subscriptions.Tracker
	shutdownMgr  *graceful.ShutdownManager

	provider provider.TransportProvider

	mu            sync.Mutex
	handlersWatch sync.WaitGroup
}

func NewNatsEvents(nc *nats.Conn, opts ...EventsOption) NatsEvents {
	// Create default options
	options := EventsOptions{
		DefaultHandlerOptions: EventHandlerOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
			JetStream: JetStreamEventOptions{
				AutoAck: true,
			},
		},
		DefaultPublishOptions: EventPublishOptions{
			Marshaller: marshaller.DefaultJsonMarshaller,
		},
		JetStreamOptions: make([]nats.JSOpt, 0),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&options)
	}

	e := &natsEventsImpl{
		nc:           nc,
		options:      options,
		lifecycleMgr: lifecycle.NewManager(),
		subsTracker:  subscriptions.NewTracker(),
		shutdownMgr:  graceful.NewShutdownManager(),
	}

	e.rootRouter = newEventRouter("", e.options.DefaultHandlerOptions)

	return e
}

// Router inherited
func (e *natsEventsImpl) Use(middlewares ...EventMiddlewareFunc) {
	e.rootRouter.Use(middlewares...)
}
func (e *natsEventsImpl) AddEventHandler(subject string, handler EventHandleFunc, opts ...EventHandlerOption) {
	e.rootRouter.AddEventHandler(subject, handler, opts...)
}
func (e *natsEventsImpl) AddEventHandlerWithMiddlewares(subject string, handler EventHandleFunc, middlewares []EventMiddlewareFunc, opts ...EventHandlerOption) {
	e.rootRouter.AddEventHandlerWithMiddlewares(subject, handler, middlewares, opts...)
}
func (e *natsEventsImpl) Group(group string) EventRouter {
	return e.rootRouter.Group(group)
}

// public methods

func (e *natsEventsImpl) Emit(ctx context.Context, subject string, payload any, opts ...EventPublishOption) error {
	if subject == "" {
		return ErrEmptySubject
	}

	publishOpts := e.mergePublishOptions(opts...)
	payloadBytes, err := publishOpts.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: payload,
	})
	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    payloadBytes,
		Header:  publishOpts.Headers,
	}

	return e.provider.Publish(ctx, msg, publishOpts)
}

func (e *natsEventsImpl) StartWithContext(ctx context.Context) error {
	handler := func(route eventInfo) nats.MsgHandler {
		return e.wrapMsgHandler(
			// ctx,
			route,
			middleware.Apply(route.handler, route.middlewares, true),
		)
	}

	for _, route := range e.rootRouter.dfs() {
		var sub provider.TransportSubscription
		var err error

		if e.options.QueueGroup != "" {
			sub, err = e.provider.QueueSubscribe(
				ctx,
				route.subject,
				e.options.QueueGroup,
				handler(route),
			)
		} else {
			sub, err = e.provider.Subscribe(
				ctx,
				route.subject,
				handler(route),
			)
		}

		if err != nil {
			e.Shutdown(ctx)

			return fmt.Errorf("failed to subscribe %s: %w", route.subject, err)
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

func (e *natsEventsImpl) Shutdown(ctx context.Context) error {
	e.subsTracker.Drain() // Drain subscriptions to stop accepting new messages

	finished := make(chan struct{})
	go func() {
		e.handlersWatch.Wait() // Wait for all active handlers to finish.
		<-finished
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

func (e *natsEventsImpl) dfs() []eventInfo {
	return e.rootRouter.dfs()
}

func (e *natsEventsImpl) wrapMsgHandler(info eventInfo, handler EventHandleFunc) nats.MsgHandler {
	return func(msg *nats.Msg) {
		if !e.shutdownMgr.StartHandler() {
			return
		}
		defer e.shutdownMgr.FinishHandler()

		handlerCtx := e.shutdownMgr.ShutdownContext()
		eventCtx := newEventContext(handlerCtx, msg, info.options.Marshaller)

		if err := handler(eventCtx); err != nil {
			e.handleHandlerError(eventCtx, info.options.JetStream)
			return
		}

		e.handleHandlerSuccess(eventCtx, info.options.JetStream)
	}
}

func (e *natsEventsImpl) handleHandlerError(eventCtx EventContext, jsOpts JetStreamEventOptions) {
	if jsOpts.Enabled {
		_ = eventCtx.Nak()
	}
}

func (e *natsEventsImpl) handleHandlerSuccess(eventCtx EventContext, jsOpts JetStreamEventOptions) {
	if jsOpts.Enabled && jsOpts.AutoAck {
		_ = eventCtx.Ack()
	}
}

func (e *natsEventsImpl) mergePublishOptions(opts ...EventPublishOption) EventPublishOptions {
	merged := e.options.DefaultPublishOptions

	// Apply functional options
	for _, opt := range opts {
		opt(&merged)
	}

	if merged.Marshaller == nil {
		merged.Marshaller = marshaller.DefaultJsonMarshaller
	}

	return merged
}
