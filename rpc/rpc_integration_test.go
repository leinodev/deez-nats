package rpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/internal/testutil"
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

	rpcServer := NewNatsRPC(nc)
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
	if err := rpcServer.CallRPC(method, addRequest{A: 10, B: 32}, &resp, CallOptions{}); err != nil {
		t.Fatalf("вызов RPC: %v", err)
	}
	if resp.Sum != 42 {
		t.Fatalf("неожиданная сумма: %d", resp.Sum)
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидалось завершение без ошибки, получили: %v", err)
	}
}

func TestRPCIntegrationCallHandlerError(t *testing.T) {
	nc := testutil.ConnectToNATS(t)

	rpcSubject := fmt.Sprintf("integration.fail.%d", time.Now().UnixNano())

	rpcServer := NewNatsRPC(nc)
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
	err := rpcServer.CallRPC(rpcSubject, addRequest{A: 1, B: 2}, &resp, CallOptions{})
	if err == nil {
		t.Fatal("ожидалась ошибка от обработчика, но CallRPC завершился успешно")
	}

	cancel()

	if err := <-startErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидалось завершение без ошибки, получили: %v", err)
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

	t.Fatal("не удалось дождаться регистрации RPC обработчиков")
}
