package events

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/internal/testutil"
	"github.com/nats-io/nats.go"
)

type sampleEvent struct {
	ID   int
	Name string
}

func TestEventsIntegrationEmitAndHandle(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	evts := NewEvents(nc)

	received := make(chan sampleEvent, 1)
	subject := fmt.Sprintf("integration.basic.%d", time.Now().UnixNano())

	evts.AddEventHandler(subject, func(ctx EventContext) error {
		var payload sampleEvent
		if err := ctx.Event(&payload); err != nil {
			t.Errorf("event deserialization failed: %v", err)
			return err
		}
		select {
		case received <- payload:
		default:
		}
		return nil
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- evts.StartWithContext(ctx)
	}()

	waitForSubscriptions(t, nc)

	want := sampleEvent{
		ID:   42,
		Name: "basic-event",
	}

	if err := evts.Emit(context.Background(), subject, want, nil); err != nil {
		t.Fatalf("event publish failed: %v", err)
	}

	select {
	case got := <-received:
		if got != want {
			t.Fatalf("unexpected payload: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestEventsIntegrationJetStream(t *testing.T) {
	nc := testutil.ConnectToNATS(t)
	js := testutil.RequireJetStream(t, nc)

	streamName := fmt.Sprintf("INTEGRATION_EVENTS_%d", time.Now().UnixNano())
	subject := fmt.Sprintf("integration.js.%d", time.Now().UnixNano())

	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.MemoryStorage,
	}); err != nil {
		t.Fatalf("jetstream stream creation failed: %v", err)
	}
	t.Cleanup(func() {
		_ = js.DeleteStream(streamName)
	})

	evts := NewEvents(nc, WithEventJetStream(js))

	received := make(chan sampleEvent, 1)
	handlerOpts := &EventHandlerOptions{
		JetStream: JetStreamEventOptions{
			Enabled: true,
			AutoAck: true,
		},
	}

	evts.AddEventHandler(subject, func(ctx EventContext) error {
		var payload sampleEvent
		if err := ctx.Event(&payload); err != nil {
			t.Errorf("event deserialization failed: %v", err)
			return err
		}
		select {
		case received <- payload:
		default:
		}
		return nil
	}, handlerOpts)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- evts.StartWithContext(ctx)
	}()

	waitForSubscriptions(t, nc)

	want := sampleEvent{
		ID:   7,
		Name: "jetstream-event",
	}

	pubOpts := &EventPublishOptions{
		JetStream: []nats.PubOpt{
			nats.MsgId(fmt.Sprintf("msg-%d", time.Now().UnixNano())),
		},
	}

	if err := evts.Emit(context.Background(), subject, want, pubOpts); err != nil {
		t.Fatalf("jetstream event publish failed: %v", err)
	}

	select {
	case got := <-received:
		if got != want {
			t.Fatalf("unexpected payload: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("jetstream handler was not invoked")
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func waitForSubscriptions(t *testing.T, nc *nats.Conn) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := nc.FlushTimeout(50 * time.Millisecond); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("failed to wait for subscriptions to register")
}
