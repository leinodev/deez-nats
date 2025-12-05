package natsrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/internal/utils"
	"github.com/nats-io/nats.go"
)

type addRequest struct {
	A int
	B int
}

type addResponse struct {
	Sum int
}

func TestRPCCallSuccess(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	rpcServer := New(nc, WithBaseRoute("myservice"))
	method := fmt.Sprintf("integration.add.%d", time.Now().UnixNano())

	rpcServer.AddUnaryHandler(method, func(ctx UnaryContext) error {
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
	if err := rpcServer.UnaryCall(ctx, "myservice."+method, addRequest{A: 10, B: 32}, &resp); err != nil {
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

func TestRPCCallError(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	rpcSubject := fmt.Sprintf("integration.fail.%d", time.Now().UnixNano())

	rpcServer := New(nc, WithBaseRoute("myservice"))
	rpcServer.AddUnaryHandler(rpcSubject, func(ctx UnaryContext) error {
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
	err := rpcServer.UnaryCall(ctx, "myservice."+rpcSubject, addRequest{A: 1, B: 2}, &resp)
	if err == nil {
		t.Fatal("expected handler error, but CallRPC succeeded")
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

func TestRPCWorkersOverflow(t *testing.T) {
	nc := utils.ConnectToNATS(t)

	rpcSubject := fmt.Sprintf("integration.sraka.%d", time.Now().UnixNano())

	rpcServer := New(nc, WithBaseRoute("myservice"), WithWorkersCount(5), WithMsgPoolSize(100))
	rpcServer.AddUnaryHandler(rpcSubject, func(ctx UnaryContext) error {
		var req addRequest
		if err := ctx.Request(&req); err != nil {
			return err
		}
		<-time.After(time.Second * 5)
		return nil
	})

	rpcServer.StartWithContext(context.Background())
	waitForRPCSubscriptions(t, nc)

	var wg sync.WaitGroup

	var failed atomic.Int32

	for range 100 {
		wg.Add(1)
		go func() {
			var resp addResponse
			err := rpcServer.UnaryCall(context.Background(), "myservice."+rpcSubject, addRequest{A: 1, B: 2}, &resp)
			if err != nil {
				if errors.Is(err, ErrPoolBusy) {
					failed.Add(1)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if failed.Load() != 95 {
		t.Error("failed, expected 95 failed")
	}
}
