package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

const (
	defaultPullBatchSize    = 1
	defaultPullFetchMaxWait = time.Second
	defaultPullRetryDelay   = 100 * time.Millisecond
)

func WithEventDefaultHandlerOptions(opts EventHandlerOptions) EventsOption {
	return func(e *natsEventsImpl) {
		e.defaultHandlerOpts = opts
		if e.defaultHandlerOpts.Marshaller == nil {
			e.defaultHandlerOpts.Marshaller = marshaller.DefaultJsonMarshaller
		}
	}
}

func WithEventDefaultPublishOptions(opts EventPublishOptions) EventsOption {
	return func(e *natsEventsImpl) {
		e.defaultPublishOpts = opts
		if e.defaultPublishOpts.Marshaller == nil {
			e.defaultPublishOpts.Marshaller = marshaller.DefaultJsonMarshaller
		}
	}
}

func WithEventJetStream(js nats.JetStreamContext) EventsOption {
	return func(e *natsEventsImpl) {
		e.js = js
	}
}

func WithEventJetStreamOptions(opts ...nats.JSOpt) EventsOption {
	return func(e *natsEventsImpl) {
		e.jsOpts = opts
	}
}

type natsEventsImpl struct {
	nc *nats.Conn

	js     nats.JetStreamContext
	jsOpts []nats.JSOpt

	defaultHandlerOpts EventHandlerOptions
	defaultPublishOpts EventPublishOptions

	rootRouter EventRouter

	mu           sync.Mutex
	subs         []*nats.Subscription
	pullWaits    []context.CancelFunc
	started      bool
	stopOnce     sync.Once
	stopWaiter   sync.WaitGroup
	shutdownFunc context.CancelFunc
}

func NewEvents(nc *nats.Conn, opts ...EventsOption) Events {
	defaultHandler := EventHandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
		JetStream: JetStreamEventOptions{
			AutoAck: true,
		},
	}
	defaultPublish := EventPublishOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}

	e := &natsEventsImpl{
		nc:                 nc,
		defaultHandlerOpts: defaultHandler,
		defaultPublishOpts: defaultPublish,
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.defaultHandlerOpts.Marshaller == nil {
		e.defaultHandlerOpts.Marshaller = marshaller.DefaultJsonMarshaller
	}
	if e.defaultPublishOpts.Marshaller == nil {
		e.defaultPublishOpts.Marshaller = marshaller.DefaultJsonMarshaller
	}

	e.rootRouter = newEventRouter("", e.defaultHandlerOpts)

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
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return errors.New("events already started")
	}
	e.started = true
	e.mu.Unlock()

	startCtx, cancel := context.WithCancel(ctx)
	e.shutdownFunc = cancel

	routes := e.rootRouter.dfs()
	if len(routes) == 0 {
		return nil
	}

	for _, info := range routes {
		if err := e.bindEvent(startCtx, info); err != nil {
			cancel()
			e.cleanupSubscriptions()
			return err
		}
	}

	select {
	case <-startCtx.Done():
		e.waitPullers()
		return startCtx.Err()
	}
}

func (e *natsEventsImpl) bindEvent(ctx context.Context, info eventInfo) error {
	handler := info.handler
	for i := len(info.middlewares) - 1; i >= 0; i-- {
		handler = info.middlewares[i](handler)
	}

	subject := info.subject
	jsOpts := append([]nats.SubOpt(nil), info.options.JetStream.SubscribeOptions...)
	if info.options.JetStream.Durable != "" {
		jsOpts = append(jsOpts, nats.Durable(info.options.JetStream.Durable))
	}
	if info.options.JetStream.SubjectTransform != nil {
		subject = info.options.JetStream.SubjectTransform(subject)
	}

	if info.options.JetStream.Enabled {
		js, err := e.ensureJetStream()
		if err != nil {
			return fmt.Errorf("jetstream context: %w", err)
		}

		msgHandler := e.wrapMsgHandler(ctx, info, handler)

		if info.options.JetStream.Pull {
			if info.options.JetStream.Durable == "" {
				return errors.New("jetstream pull consumer requires durable name")
			}
			sub, err := js.PullSubscribe(subject, info.options.JetStream.Durable, jsOpts...)
			if err != nil {
				return fmt.Errorf("pull subscribe %s: %w", subject, err)
			}
			e.trackSubscription(sub)

			pullCtx, cancel := context.WithCancel(ctx)
			e.trackPull(cancel)
			e.stopWaiter.Add(1)
			go func(batch int, expire time.Duration) {
				defer e.stopWaiter.Done()
				for {
					select {
					case <-pullCtx.Done():
						return
					default:
					}

					msgs, err := sub.Fetch(batch, nats.MaxWait(expire))
					if err != nil {
						if errors.Is(err, nats.ErrTimeout) {
							continue
						}
						if errors.Is(err, context.Canceled) {
							return
						}

						// retry after small delay
						time.Sleep(defaultPullRetryDelay)
						continue
					}

					for _, msg := range msgs {
						msgHandler(msg)
					}
				}
			}(maxInt(info.options.JetStream.PullBatch, defaultPullBatchSize), defaultDuration(info.options.JetStream.PullExpire, defaultPullFetchMaxWait))
		} else {
			var sub *nats.Subscription
			if info.options.JetStream.DeliverGroup != "" {
				sub, err = js.QueueSubscribe(subject, info.options.JetStream.DeliverGroup, msgHandler, jsOpts...)
			} else {
				sub, err = js.Subscribe(subject, msgHandler, jsOpts...)
			}
			if err != nil {
				return fmt.Errorf("subscribe %s: %w", subject, err)
			}
			e.trackSubscription(sub)
		}

		return nil
	}

	msgHandler := e.wrapMsgHandler(ctx, info, handler)
	var (
		sub *nats.Subscription
		err error
	)
	if info.options.Queue != "" {
		sub, err = e.nc.QueueSubscribe(subject, info.options.Queue, msgHandler)
	} else {
		sub, err = e.nc.Subscribe(subject, msgHandler)
	}
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}

	e.trackSubscription(sub)
	return nil
}

func (e *natsEventsImpl) wrapMsgHandler(ctx context.Context, info eventInfo, handler EventHandleFunc) nats.MsgHandler {
	return func(msg *nats.Msg) {
		eventCtx := newEventContext(ctx, msg, info.options.Marshaller)

		err := handler(eventCtx)
		if err != nil {
			if info.options.JetStream.Enabled {
				_ = eventCtx.Nak()
			}
			return
		}

		if info.options.JetStream.Enabled && info.options.JetStream.AutoAck {
			_ = eventCtx.Ack()
		}
	}
}

func (e *natsEventsImpl) Emit(ctx context.Context, subject string, payload any, opts *EventPublishOptions) error {
	if subject == "" {
		return errors.New("empty subject")
	}

	options := e.defaultPublishOpts
	if opts != nil {
		options.Headers = cloneMap(opts.Headers)
		options.JetStream = append([]nats.PubOpt(nil), opts.JetStream...)
		if opts.Marshaller != nil {
			options.Marshaller = opts.Marshaller
		}
	}

	if options.Marshaller == nil {
		options.Marshaller = marshaller.DefaultJsonMarshaller
	}

	payloadBytes, err := options.Marshaller.Marshall(&marshaller.MarshalObject{
		Data: payload,
	})
	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    payloadBytes,
	}

	for k, v := range options.Headers {
		if msg.Header == nil {
			msg.Header = nats.Header{}
		}
		msg.Header.Add(k, v)
	}

	if len(options.JetStream) > 0 {
		js, err := e.ensureJetStream()
		if err != nil {
			return err
		}

		_, err = js.PublishMsg(msg, options.JetStream...)
		return err
	}

	return e.nc.PublishMsg(msg)
}

func (e *natsEventsImpl) ensureJetStream() (nats.JetStreamContext, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.js != nil {
		return e.js, nil
	}

	js, err := e.nc.JetStream(e.jsOpts...)
	if err != nil {
		return nil, err
	}

	e.js = js
	return js, nil
}

func (e *natsEventsImpl) trackSubscription(sub *nats.Subscription) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subs = append(e.subs, sub)
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
	e.cleanupSubscriptions()
}

func (e *natsEventsImpl) cleanupSubscriptions() {
	e.mu.Lock()
	subs := e.subs
	e.subs = nil
	e.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Drain()
	}
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

func cloneMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
