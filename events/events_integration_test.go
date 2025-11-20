package events

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/internal/utils"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type sampleEvent struct {
	ID   int
	Name string
}

func TestEventsIntegrationEmitAndHandle(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	evts := NewCoreEvents(nc)

	received := make(chan sampleEvent, 1)
	subject := fmt.Sprintf("integration.basic.%d", time.Now().UnixNano())

	evts.AddEventHandler(subject, func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
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
	}, func(opts *CoreEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	})

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

	if err := evts.Emit(context.Background(), subject, want, func(opts *CoreEventEmitOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	}); err != nil {
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
	nc := utils.ConnectToNATS(t)
	jsCtx := utils.RequireJetStream(t, nc)

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("failed to create jetstream: %v", err)
	}

	streamName := fmt.Sprintf("INTEGRATION_EVENTS_%d", time.Now().UnixNano())
	subject := fmt.Sprintf("integration.js.%d", time.Now().UnixNano())

	if _, err := jsCtx.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.MemoryStorage,
	}); err != nil {
		t.Fatalf("jetstream stream creation failed: %v", err)
	}
	t.Cleanup(func() {
		_ = jsCtx.DeleteStream(streamName)
	})

	evts := NewJetStreamEvents(js, func(opts *JetStreamEventsOptions) {
		opts.Stream = streamName
		opts.DeliverGroup = "test-group"
	})

	received := make(chan sampleEvent, 1)

	evts.AddEventHandler(subject, func(ctx EventContext[jetstream.Msg, any]) error {
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
	}, func(opts *JetStreamEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	})

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

	if err := evts.Emit(context.Background(), subject, want, func(opts *JetStreamEventEmitOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	}); err != nil {
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

func TestEventsIntegrationGracefulShutdown(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	evts := NewCoreEvents(nc)

	received := make(chan sampleEvent, 1)
	subject := fmt.Sprintf("integration.shutdown.%d", time.Now().UnixNano())

	handlerStarted := make(chan struct{})
	handlerFinished := make(chan struct{})
	shutdownComplete := make(chan struct{})

	evts.AddEventHandler(subject, func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		close(handlerStarted)
		// Simulate long processing
		time.Sleep(500 * time.Millisecond)
		var payload sampleEvent
		if err := ctx.Event(&payload); err != nil {
			t.Errorf("event deserialization failed: %v", err)
			return err
		}
		select {
		case received <- payload:
		default:
		}
		close(handlerFinished)
		return nil
	}, func(opts *CoreEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- evts.StartWithContext(ctx)
	}()

	waitForSubscriptions(t, nc)

	want := sampleEvent{
		ID:   42,
		Name: "shutdown-test",
	}

	// Send event
	if err := evts.Emit(context.Background(), subject, want, func(opts *CoreEventEmitOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	}); err != nil {
		t.Fatalf("event publish failed: %v", err)
	}

	// Wait for handler to start
	<-handlerStarted

	// Initiate graceful shutdown
	go func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		cancel() // Cancel context to stop accepting new messages
		err := evts.Shutdown(shutdownCtx)
		if err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
		close(shutdownComplete)
	}()

	// Wait for handler to finish
	select {
	case <-handlerFinished:
		// Handler finished successfully
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish within timeout")
	}

	// Wait for graceful shutdown to complete
	select {
	case <-shutdownComplete:
		// Shutdown completed successfully
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not complete within timeout")
	}

	// Verify that event was processed
	select {
	case got := <-received:
		if got != want {
			t.Fatalf("unexpected payload: %#v", got)
		}
	default:
		t.Fatal("event was not processed")
	}

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestEventsIntegrationGracefulShutdownTimeout(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	evts := NewCoreEvents(nc)

	subject := fmt.Sprintf("integration.shutdown.timeout.%d", time.Now().UnixNano())

	handlerStarted := make(chan struct{})

	evts.AddEventHandler(subject, func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		close(handlerStarted)
		// Simulate very long processing that will exceed shutdown timeout
		time.Sleep(2 * time.Second)
		var payload sampleEvent
		if err := ctx.Event(&payload); err != nil {
			return err
		}
		return nil
	}, func(opts *CoreEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- evts.StartWithContext(ctx)
	}()

	waitForSubscriptions(t, nc)

	want := sampleEvent{
		ID:   42,
		Name: "shutdown-timeout-test",
	}

	// Send event
	if err := evts.Emit(context.Background(), subject, want, func(opts *CoreEventEmitOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	}); err != nil {
		t.Fatalf("event publish failed: %v", err)
	}

	// Wait for handler to start
	<-handlerStarted

	// Initiate graceful shutdown with short timeout
	cancel() // Cancel context to stop accepting new messages
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shutdownCancel()

	err := evts.Shutdown(shutdownCtx)
	if err == nil {
		t.Fatal("expected shutdown to timeout, but it completed")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded error, got: %v", err)
	}

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}
