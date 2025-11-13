package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/internal/testutil"
	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
)

type addRequest struct {
	A int
	B int
}

type addResponse struct {
	Sum int
}

func TestRPCIntegrationCallSuccess(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	method := fmt.Sprintf("integration.add.%d", time.Now().UnixNano())

	rpcServer.AddRPCHandler(method, func(ctx RPCContext) error {
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		return ctx.Ok(&addResponse{Sum: req.A + req.B})
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 10, B: 32}, &resp); err != nil {
		t.Fatalf("rpc call failed: %v", err)
	}
	if resp.Sum != 42 {
		t.Fatalf("unexpected sum: %d", resp.Sum)
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestRPCIntegrationCallHandlerError(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	rpcSubject := fmt.Sprintf("integration.fail.%d", time.Now().UnixNano())

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	rpcServer.AddRPCHandler(rpcSubject, func(ctx RPCContext) error {
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		return errors.New("integration handler failure")
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	err := rpcServer.CallRPC(ctx, "myservice."+rpcSubject, addRequest{A: 1, B: 2}, &resp)
	if err == nil {
		t.Fatal("expected handler error, but CallRPC succeeded")
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestRPCIntegrationTypedCallSuccess(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	method := fmt.Sprintf("integration.typed.add.%d", time.Now().UnixNano())

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	AddTypedRPCHandler(rpcServer, method, func(ctx RPCContext, request addRequest) (addResponse, error) {
		return addResponse{Sum: request.A + request.B}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 5, B: 7}, &resp); err != nil {
		t.Fatalf("typed RPC call failed: %v", err)
	}
	if resp.Sum != 12 {
		t.Fatalf("unexpected sum: %d", resp.Sum)
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestRPCIntegrationTypedCallHandlerError(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	method := fmt.Sprintf("integration.typed.fail.%d", time.Now().UnixNano())

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	AddTypedRPCHandler(rpcServer, method, func(ctx RPCContext, request addRequest) (addResponse, error) {
		return addResponse{}, errors.New("typed handler failure")
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 1, B: 2}, &resp)
	if err == nil {
		t.Fatal("expected error from typed handler, but CallRPC succeeded")
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestRPCIntegrationTypedCallWithCustomMarshaller(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	method := fmt.Sprintf("integration.typed.marshaller.%d", time.Now().UnixNano())

	recMarshaller := newRecordingMarshaller(nil)

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	AddTypedRPCHandlerWithMarshaller(rpcServer, method, func(ctx RPCContext, request addRequest) (addResponse, error) {
		return addResponse{Sum: request.A + request.B}, nil
	}, recMarshaller)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 20, B: 22}, &resp, WithCallMarshaller(recMarshaller)); err != nil {
		t.Fatalf("typed RPC call with custom marshaller failed: %v", err)
	}
	if resp.Sum != 42 {
		t.Fatalf("unexpected sum: %d", resp.Sum)
	}

	marshalCount, unmarshalCount := recMarshaller.counts()
	if marshalCount < 2 {
		t.Fatalf("expected at least two Marshall calls, got: %d", marshalCount)
	}
	if unmarshalCount < 2 {
		t.Fatalf("expected at least two Unmarshall calls, got: %d", unmarshalCount)
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func waitForRPCSubscriptions(t *testing.T, nc *nats.Conn) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := nc.FlushTimeout(50 * time.Millisecond); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("failed to wait for RPC handlers registration")
}

type recordingMarshaller struct {
	inner marshaller.PayloadMarshaller

	mu             sync.Mutex
	marshalCount   int
	unmarshalCount int
}

func newRecordingMarshaller(inner marshaller.PayloadMarshaller) *recordingMarshaller {
	if inner == nil {
		inner = marshaller.DefaultJsonMarshaller
	}
	return &recordingMarshaller{inner: inner}
}

func (r *recordingMarshaller) Marshall(v *marshaller.MarshalObject) ([]byte, error) {
	r.mu.Lock()
	r.marshalCount++
	r.mu.Unlock()
	return r.inner.Marshall(v)
}

func (r *recordingMarshaller) Unmarshall(data []byte, v *marshaller.MarshalObject) error {
	r.mu.Lock()
	r.unmarshalCount++
	r.mu.Unlock()
	return r.inner.Unmarshall(data, v)
}

func (r *recordingMarshaller) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.marshalCount, r.unmarshalCount
}

func TestRPCIntegrationGracefulShutdown(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	method := fmt.Sprintf("integration.shutdown.%d", time.Now().UnixNano())

	handlerStarted := make(chan struct{})
	handlerFinished := make(chan struct{})
	shutdownComplete := make(chan struct{})

	rpcServer.AddRPCHandler(method, func(ctx RPCContext) error {
		close(handlerStarted)
		// Simulate long processing
		time.Sleep(500 * time.Millisecond)
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		close(handlerFinished)
		return ctx.Ok(&addResponse{Sum: req.A + req.B})
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	// Send request in a separate goroutine
	var resp addResponse
	callDone := make(chan error, 1)
	go func() {
		callDone <- rpcServer.CallRPC(context.Background(), "myservice."+method, addRequest{A: 10, B: 32}, &resp)
	}()

	// Wait for handler to start
	<-handlerStarted

	// Initiate graceful shutdown
	go func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		cancel() // Cancel context to stop accepting new messages
		err := rpcServer.Shutdown(shutdownCtx)
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

	// Verify that request was processed
	select {
	case err := <-callDone:
		if err != nil {
			t.Fatalf("rpc call failed: %v", err)
		}
		if resp.Sum != 42 {
			t.Fatalf("unexpected sum: %d", resp.Sum)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("rpc call did not complete")
	}

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected completion without error, got: %v", err)
	}
}

func TestRPCIntegrationGracefulShutdownTimeout(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	rpcServer := NewNatsRPC(nc, WithBaseRoute("myservice"))
	method := fmt.Sprintf("integration.shutdown.timeout.%d", time.Now().UnixNano())

	handlerStarted := make(chan struct{})

	rpcServer.AddRPCHandler(method, func(ctx RPCContext) error {
		close(handlerStarted)
		// Simulate very long processing that will exceed shutdown timeout
		time.Sleep(2 * time.Second)
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		return ctx.Ok(&addResponse{Sum: req.A + req.B})
	})

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	// Send request in a separate goroutine
	var resp addResponse
	go func() {
		_ = rpcServer.CallRPC(context.Background(), "myservice."+method, addRequest{A: 10, B: 32}, &resp)
	}()

	// Wait for handler to start
	<-handlerStarted

	// Initiate graceful shutdown with short timeout
	cancel() // Cancel context to stop accepting new messages
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shutdownCancel()

	err := rpcServer.Shutdown(shutdownCtx)
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
