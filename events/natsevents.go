package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	subscriptionMgr *subscriptions.Manager

	mu           sync.Mutex
	pullWaits    []context.CancelFunc
	stopWaiter   sync.WaitGroup
	shutdownFunc context.CancelFunc
}

func NewNatsEvents(nc *nats.Conn, opts *EventsOptions) NatsEvents {
	if opts == nil {
		defaultOpts := NewEventsOptionsBuilder().Build()
		opts = &defaultOpts
	}

	e := &natsEventsImpl{
		nc:              nc,
		options:         *opts,
		lifecycleMgr:    lifecycle.NewManager(),
		subscriptionMgr: subscriptions.NewManager(),
	}

	e.rootRouter = newEventRouter("", e.options.DefaultHandlerOptions)

	return e
}

// Router inherited
func (e *natsEventsImpl) Use(middlewares ...EventMiddlewareFunc) {
	e.rootRouter.Use(middlewares...)
}

func (e *natsEventsImpl) AddEventHandler(subject string, handler EventHandleFunc, opts *EventHandlerOptions, middlewares ...EventMiddlewareFunc) {
	e.rootRouter.AddEventHandler(subject, handler, opts, middlewares...)
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

	startCtx, cancel := context.WithCancel(ctx)
	e.shutdownFunc = cancel
	defer cancel()

	if err := e.bindAllEvents(startCtx); err != nil {
		e.subscriptionMgr.Cleanup()
		return err
	}

	<-startCtx.Done()
	e.waitPullers()
	return startCtx.Err()
}

func (e *natsEventsImpl) bindAllEvents(ctx context.Context) error {
	routes := e.rootRouter.dfs()
	if len(routes) == 0 {
		return nil
	}

	for _, info := range routes {
		if err := e.bindEvent(ctx, info); err != nil {
			return err
		}
	}

	return nil
}

func (e *natsEventsImpl) bindEvent(ctx context.Context, info eventInfo) error {
	handler := middleware.Apply(info.handler, info.middlewares, true)
	subject := info.subject
	if info.options.JetStream.SubjectTransform != nil {
		subject = info.options.JetStream.SubjectTransform(subject)
	}

	if info.options.JetStream.Enabled {
		return e.bindJetStreamEvent(ctx, info, handler, subject)
	}

	return e.bindStandardEvent(ctx, info, handler, subject)
}

func (e *natsEventsImpl) bindJetStreamEvent(ctx context.Context, info eventInfo, handler EventHandleFunc, subject string) error {
	js, err := e.ensureJetStream()
	if err != nil {
		return fmt.Errorf("jetstream context: %w", err)
	}

	jsOpts := e.buildJetStreamSubscribeOptions(info.options.JetStream)
	msgHandler := e.wrapMsgHandler(ctx, info, handler)

	if info.options.JetStream.Pull {
		return e.bindPullConsumer(ctx, js, subject, info.options.JetStream, jsOpts, msgHandler)
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

func (e *natsEventsImpl) bindPullConsumer(ctx context.Context, js nats.JetStreamContext, subject string, jsOpts JetStreamEventOptions, subscribeOpts []nats.SubOpt, msgHandler nats.MsgHandler) error {
	if jsOpts.Durable == "" {
		return ErrJetStreamPullRequiresDurable
	}

	sub, err := js.PullSubscribe(subject, jsOpts.Durable, subscribeOpts...)
	if err != nil {
		return fmt.Errorf("pull subscribe %s: %w", subject, err)
	}

	e.subscriptionMgr.Track(sub)
	e.startPullConsumer(ctx, sub, jsOpts, msgHandler)
	return nil
}

func (e *natsEventsImpl) startPullConsumer(ctx context.Context, sub *nats.Subscription, jsOpts JetStreamEventOptions, msgHandler nats.MsgHandler) {
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

func (e *natsEventsImpl) bindStandardEvent(ctx context.Context, info eventInfo, handler EventHandleFunc, subject string) error {
	msgHandler := e.wrapMsgHandler(ctx, info, handler)

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

func (e *natsEventsImpl) wrapMsgHandler(ctx context.Context, info eventInfo, handler EventHandleFunc) nats.MsgHandler {
	return func(msg *nats.Msg) {
		eventCtx := newEventContext(ctx, msg, info.options.Marshaller)

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

func (e *natsEventsImpl) Emit(ctx context.Context, subject string, payload any, opts *EventPublishOptions) error {
	if subject == "" {
		return ErrEmptySubject
	}

	publishOpts := e.mergePublishOptions(opts)
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

func (e *natsEventsImpl) mergePublishOptions(opts *EventPublishOptions) EventPublishOptions {
	merged := e.options.DefaultPublishOptions

	if opts != nil {
		if opts.Headers != nil {
			merged.Headers = opts.Headers
		}
		if len(opts.JetStream) > 0 {
			merged.JetStream = append([]nats.PubOpt(nil), opts.JetStream...)
		}
		if opts.Marshaller != nil {
			merged.Marshaller = opts.Marshaller
		}
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
	e.subscriptionMgr.Cleanup()
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
