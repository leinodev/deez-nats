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

	opts := NewRPCOptionsBuilder().WithBaseRoute("myservice").Build()
	rpcServer := NewNatsRPC(nc, &opts)
	method := fmt.Sprintf("integration.add.%d", time.Now().UnixNano())

	rpcServer.AddRPCHandler(method, func(ctx RPCContext) error {
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		return ctx.Ok(&addResponse{Sum: req.A + req.B})
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 10, B: 32}, &resp, CallOptions{}); err != nil {
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

	opts := NewRPCOptionsBuilder().WithBaseRoute("myservice").Build()
	rpcServer := NewNatsRPC(nc, &opts)
	rpcServer.AddRPCHandler(rpcSubject, func(ctx RPCContext) error {
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		return errors.New("integration handler failure")
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	err := rpcServer.CallRPC(ctx, "myservice."+rpcSubject, addRequest{A: 1, B: 2}, &resp, CallOptions{})
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

	opts := NewRPCOptionsBuilder().WithBaseRoute("myservice").Build()
	rpcServer := NewNatsRPC(nc, &opts)
	AddTypedRPCHandler(rpcServer, method, func(ctx RPCContext, request addRequest) (addResponse, error) {
		return addResponse{Sum: request.A + request.B}, nil
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 5, B: 7}, &resp, CallOptions{}); err != nil {
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

	opts := NewRPCOptionsBuilder().WithBaseRoute("myservice").Build()
	rpcServer := NewNatsRPC(nc, &opts)
	AddTypedRPCHandler(rpcServer, method, func(ctx RPCContext, request addRequest) (addResponse, error) {
		return addResponse{}, errors.New("typed handler failure")
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)
	go func() {
		startErr <- rpcServer.StartWithContext(ctx)
	}()

	waitForRPCSubscriptions(t, nc)

	var resp addResponse
	err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 1, B: 2}, &resp, CallOptions{})
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

	opts := NewRPCOptionsBuilder().WithBaseRoute("myservice").Build()
	rpcServer := NewNatsRPC(nc, &opts)
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
	if err := rpcServer.CallRPC(ctx, "myservice."+method, addRequest{A: 20, B: 22}, &resp, CallOptions{Marshaller: recMarshaller}); err != nil {
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
