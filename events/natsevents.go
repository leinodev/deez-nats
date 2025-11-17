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

	lifecycleMgr    *lifecycle.Manager
	subscriptionMgr *subscriptions.Tracker
	shutdownMgr     *graceful.ShutdownManager

	mu           sync.Mutex
	pullWaits    []context.CancelFunc
	stopWaiter   sync.WaitGroup
	shutdownFunc context.CancelFunc
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
		nc:              nc,
		options:         options,
		lifecycleMgr:    lifecycle.NewManager(),
		subscriptionMgr: subscriptions.NewTracker(),
		shutdownMgr:     graceful.NewShutdownManager(),
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

func (e *natsEventsImpl) dfs() []eventInfo {
	return e.rootRouter.dfs()
}

func (e *natsEventsImpl) StartWithContext(ctx context.Context) error {
	if err := e.lifecycleMgr.MarkAsStarted(); err != nil {
		return fmt.Errorf("events: %w", err)
	}

	// Set parent context for shutdown manager
	e.shutdownMgr.SetParentContext(ctx)

	startCtx, cancel := context.WithCancel(ctx)
	e.shutdownFunc = cancel
	defer cancel()

	if err := e.bindAllEvents(); err != nil {
		e.subscriptionMgr.Drain()
		return err
	}

	<-startCtx.Done()
	e.waitPullers()
	_ = e.Shutdown(context.Background())
	return startCtx.Err()
}

func (e *natsEventsImpl) bindAllEvents() error {
	routes := e.rootRouter.dfs()
	if len(routes) == 0 {
		return nil
	}

	for _, info := range routes {
		if err := e.bindEvent(info); err != nil {
			return err
		}
	}

	return nil
}

func (e *natsEventsImpl) bindEvent(info eventInfo) error {
	handler := middleware.Apply(info.handler, info.middlewares, true)
	subject := info.subject
	if info.options.JetStream.SubjectTransform != nil {
		subject = info.options.JetStream.SubjectTransform(subject)
	}

	if info.options.JetStream.Enabled {
		return e.bindJetStreamEvent(info, handler, subject)
	}

	return e.bindStandardEvent(info, handler, subject)
}

func (e *natsEventsImpl) bindJetStreamEvent(info eventInfo, handler EventHandleFunc, subject string) error {
	js, err := e.ensureJetStream()
	if err != nil {
		return fmt.Errorf("jetstream context: %w", err)
	}

	jsOpts := e.buildJetStreamSubscribeOptions(info.options.JetStream)
	msgHandler := e.wrapMsgHandler(info, handler)

	if info.options.JetStream.Pull {
		return e.bindPullConsumer(js, subject, info.options.JetStream, jsOpts, msgHandler)
	}

	return e.bindPushConsumer(js, subject, info.options.JetStream, jsOpts, msgHandler)
}

func (e *natsEventsImpl) buildJetStreamSubscribeOptions(jsOpts JetStreamEventOptions) []nats.SubOpt {
	opts := append([]nats.SubOpt(nil), jsOpts.SubscribeOptions...)
	if jsOpts.Durable != "" {
		opts = append(opts, nats.Durable(jsOpts.Durable))
	}
	return opts
}

func (e *natsEventsImpl) bindPullConsumer(js nats.JetStreamContext, subject string, jsOpts JetStreamEventOptions, subscribeOpts []nats.SubOpt, msgHandler nats.MsgHandler) error {
	if jsOpts.Durable == "" {
		return ErrJetStreamPullRequiresDurable
	}

	sub, err := js.PullSubscribe(subject, jsOpts.Durable, subscribeOpts...)
	if err != nil {
		return fmt.Errorf("pull subscribe %s: %w", subject, err)
	}

	e.subscriptionMgr.Track(sub)
	e.startPullConsumer(sub, jsOpts, msgHandler)
	return nil
}

func (e *natsEventsImpl) startPullConsumer(sub *nats.Subscription, jsOpts JetStreamEventOptions, msgHandler nats.MsgHandler) {
	ctx := e.shutdownMgr.ShutdownContext()
	pullCtx, cancel := context.WithCancel(ctx)
	e.trackPull(cancel)
	e.stopWaiter.Add(1)

	batch := maxInt(jsOpts.PullBatch, 1)
	expire := defaultDuration(jsOpts.PullExpire, time.Second)
	retryDelay := defaultDuration(jsOpts.PullRetryDelay, 100*time.Millisecond)

	go func() {
		defer e.stopWaiter.Done()
		e.runPullConsumer(pullCtx, sub, batch, expire, retryDelay, msgHandler)
	}()
}

func (e *natsEventsImpl) runPullConsumer(ctx context.Context, sub *nats.Subscription, batch int, expire time.Duration, retryDelay time.Duration, msgHandler nats.MsgHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := sub.Fetch(batch, nats.MaxWait(expire))
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			time.Sleep(retryDelay)
			continue
		}

		for _, msg := range msgs {
			msgHandler(msg)
		}
	}
}

func (e *natsEventsImpl) bindPushConsumer(js nats.JetStreamContext, subject string, jsOpts JetStreamEventOptions, subscribeOpts []nats.SubOpt, msgHandler nats.MsgHandler) error {
	var sub *nats.Subscription
	var err error

	if jsOpts.DeliverGroup != "" {
		sub, err = js.QueueSubscribe(subject, jsOpts.DeliverGroup, msgHandler, subscribeOpts...)
	} else {
		sub, err = js.Subscribe(subject, msgHandler, subscribeOpts...)
	}

	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}

	e.subscriptionMgr.Track(sub)
	return nil
}

func (e *natsEventsImpl) bindStandardEvent(info eventInfo, handler EventHandleFunc, subject string) error {
	msgHandler := e.wrapMsgHandler(info, handler)

	var sub *nats.Subscription
	var err error

	if info.options.Queue != "" {
		sub, err = e.nc.QueueSubscribe(subject, info.options.Queue, msgHandler)
	} else {
		sub, err = e.nc.Subscribe(subject, msgHandler)
	}

	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}

	e.subscriptionMgr.Track(sub)
	return nil
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

	return e.publishMessage(ctx, msg, publishOpts)
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

func (e *natsEventsImpl) publishMessage(ctx context.Context, msg *nats.Msg, opts EventPublishOptions) error {
	if len(opts.JetStream) > 0 {
		return e.publishToJetStream(ctx, msg, opts.JetStream)
	}

	return e.nc.PublishMsg(msg)
}

func (e *natsEventsImpl) publishToJetStream(ctx context.Context, msg *nats.Msg, jsOpts []nats.PubOpt) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled: %w", err)
	}

	js, err := e.ensureJetStream()
	if err != nil {
		return fmt.Errorf("ensure jetstream: %w", err)
	}

	_, err = js.PublishMsg(msg, jsOpts...)
	return err
}

func (e *natsEventsImpl) ensureJetStream() (nats.JetStreamContext, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.options.JetStream != nil {
		return e.options.JetStream, nil
	}

	return e.nc.JetStream(e.options.JetStreamOptions...)
}

func (e *natsEventsImpl) trackPull(cancel context.CancelFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.pullWaits = append(e.pullWaits, cancel)
}

func (e *natsEventsImpl) waitPullers() {
	e.mu.Lock()
	pulls := e.pullWaits
	e.pullWaits = nil
	e.mu.Unlock()

	for _, cancel := range pulls {
		cancel()
	}

	e.stopWaiter.Wait()
}

func (e *natsEventsImpl) Shutdown(ctx context.Context) error {
	// Stop pull consumers first
	e.waitPullers()

	// Drain subscriptions to stop accepting new messages
	e.subscriptionMgr.Drain()

	// Wait for all active handlers to finish
	if err := e.shutdownMgr.Shutdown(ctx); err != nil {
		return err
	}

	// Unsubscribe from all routes
	e.subscriptionMgr.Unsubscribe()
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func defaultDuration(d time.Duration, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}
