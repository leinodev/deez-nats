# deez-nats

_[Русская версия документации](README.ru.md)_

Utilities for building RPC and event-driven applications on top of [NATS](https://nats.io) in Go. The library ships with a unified router for RPC methods and events, middleware support, typed helpers, and several marshallers (JSON and Protobuf) to speed up onboarding and simplify long-term maintenance.

## Features

- **Unified router** with grouping (`Group`) and middleware inheritance for both RPC and events.
- **RPC handling** with automatic ack/nak management and a convenient `RPCContext` for reading requests, sending responses, and working with headers.
- **Event handlers with JetStream**: queue / pull consumers, auto-ack, durable configuration, and subject transforms.
- **Typed helpers** for RPC (`rpc.AddTyped…`) and events (`events.AddTyped…`) with generics support and pluggable marshallers.
- **Flexible marshallers**: built-in JSON and Protobuf implementations, plus the option to provide your own.
- **Examples** for a quick start: from a simple scenario to a fully typed pipeline.

## Installation

```sh
go get github.com/leinodev/deez-nats
```

Requires Go 1.21+ (the module is built with `go 1.25`).

## Quick Start

```go
nc, _ := nats.Connect(nats.DefaultURL)
defer nc.Close()

eventsSvc := events.NewNatsEvents(nc)
rpcSvc := rpc.NewNatsRPC(nc)

rpcSvc.AddRPCHandler("user.ping", func(ctx rpc.RPCContext) error {
    var req PingRequest
    if err := ctx.Request(&req); err != nil {
        return err
    }
    return ctx.Ok(PingResponse{Message: "pong"})
})

eventsSvc.AddEventHandler("user.created", func(ctx events.EventContext) error {
    var payload UserCreatedEvent
    if err := ctx.Event(&payload); err != nil {
        return err
    }
    fmt.Printf("created: %#v\n", payload)
    return nil
})

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go rpcSvc.StartWithContext(ctx)
go eventsSvc.StartWithContext(ctx)

_ = eventsSvc.Emit(ctx, "user.created", UserCreatedEvent{ID: "42"})
var resp PingResponse
_ = rpcSvc.CallRPC(ctx, "user.ping", PingRequest{ID: "42"}, &resp)
```

## RPC

- `rpc.NewNatsRPC` creates a service with the default JSON marshaller.
- `AddRPCHandler` builds a hierarchical route tree; methods can be grouped (`Service.Group("user")`).
- `RPCContext` offers:
  - `Request(&reqStruct)` — request deserialization;
  - `Ok(response)` — sending a successful response;
  - `Headers()` and `RequestHeaders()` — working with headers.
- `CallRPC` wraps a request with NATS timeout handling and response deserialization.
- For generics, use `rpc.AddTypedJsonRPCHandler`, `AddTypedProtoRPCHandler`, or `AddTypedRPCHandler` with your custom marshaller.
- Use `rpc.WithHandlerMiddlewares(...)` to add middlewares to specific handlers.

### RPC Options

Use functional options to configure RPC service:

```go
rpcSvc := rpc.NewNatsRPC(nc,
    rpc.WithBaseRoute("myservice"),
    rpc.WithDefaultHandlerMarshaller(customMarshaller),
    rpc.WithDefaultCallOptions(
        rpc.WithCallHeader("X-Service", "my-service"),
    ),
)

// Add handler with middlewares
rpcSvc.AddRPCHandler("user.get", handler,
    rpc.WithHandlerMarshaller(customMarshaller),
    rpc.WithHandlerMiddlewares(
        authMiddleware,
        loggingMiddleware,
    ),
)

// Call RPC with options
rpcSvc.CallRPC(ctx, "user.ping", request, &response,
    rpc.WithCallHeader("X-Request-ID", "123"),
    rpc.WithCallMarshaller(customMarshaller),
)
```

## Events

- `events.NewNatsEvents` creates a service that supports standard subscriptions and JetStream.
- `AddEventHandler` lets you specify a queue (`Queue`) and JetStream options: durable, pull, deliver group, subject transform, etc.
- For pull consumers, set `JetStream.Pull = true`, `PullBatch`, `PullExpire`, and `Durable`.
- `EventContext` provides:
  - `Event(&payload)` — message deserialization;
  - `Ack/Nak/Term/InProgress` — JetStream delivery control;
  - access to `Headers()` and the original `*nats.Msg`.
- Emit events with `Emit(ctx, subject, payload, opts...)`; you can include headers and JetStream options via functional options.
- Typed helpers: `events.AddTypedEventHandler`, `AddTypedJsonEventHandler`, `AddTypedProtoEventHandler`.

### Events Options

Use functional options to configure event service:

```go
eventsSvc := events.NewNatsEvents(nc,
    events.WithJetStreamContext(js),
    events.WithTimeout(time.Second),
    events.WithJetStream(true),
    events.WithAutoAck(true),
    events.WithDefaultHandlerOptions(
        events.WithHandlerQueue("my-queue"),
        events.WithHandlerJetStream(
            events.WithJSEnabled(true),
            events.WithJSAutoAck(true),
            events.WithJSDurable("my-consumer"),
        ),
    ),
    events.WithDefaultPublishOptions(
        events.WithPublishMarshaller(customMarshaller),
        events.WithPublishHeader("X-Source", "my-service"),
    ),
)

// Add handler with JetStream options
eventsSvc.AddEventHandler("entity.created", handler,
    events.WithHandlerJetStream(
        events.WithJSEnabled(true),
        events.WithJSAutoAck(true),
        events.WithJSDurable("entity-created-consumer"),
        events.WithJSDeliverGroup("entity-events"),
    ),
)

// Emit event with options
eventsSvc.Emit(ctx, "user.created", payload,
    events.WithPublishHeader("X-Source", "my-service"),
    events.WithPublishJetStreamOptions(nats.MsgId("msg-id")),
)
```

## Graceful Shutdown

The library supports graceful shutdown for both RPC and event services. This allows you to properly finish processing active requests and events before stopping the service.

### Usage

Both services (`rpc.NatsRPC` and `events.NatsEvents`) implement the `Shutdown(ctx context.Context) error` method, which:

1. **Stops accepting new messages** — subscriptions are drained, new requests and events are no longer accepted.
2. **Waits for active handlers to finish** — all running handlers are given a chance to complete their work.
3. **Unsubscribes from all subscriptions** — after all handlers finish, unsubscription from NATS subjects occurs.

### Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/leinodev/deez-nats/events"
    "github.com/leinodev/deez-nats/rpc"
)

func main() {
    nc, _ := nats.Connect(nats.DefaultURL)
    defer nc.Close()

    eventService := events.NewNatsEvents(nc)
    rpcService := rpc.NewNatsRPC(nc)

    // ... configure handlers ...

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go eventService.StartWithContext(ctx)
    go rpcService.StartWithContext(ctx)

    // Wait for shutdown signal (SIGINT or SIGTERM)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    <-sigChan
    log.Println("Received shutdown signal, starting graceful shutdown...")

    // Cancel context to stop processing new messages
    cancel()

    // Perform graceful shutdown with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()

    if err := eventService.Shutdown(shutdownCtx); err != nil {
        log.Printf("events service shutdown error: %v", err)
    } else {
        log.Println("events service shutdown completed")
    }

    if err := rpcService.Shutdown(shutdownCtx); err != nil {
        log.Printf("rpc service shutdown error: %v", err)
    } else {
        log.Println("rpc service shutdown completed")
    }

    log.Println("Graceful shutdown completed")
}
```

### Important Notes

- **Timeout**: Always use a context with timeout for `Shutdown()` to avoid infinite waiting.
- **Order**: First cancel the context (`cancel()`), then call `Shutdown()` — this ensures that new messages won't be accepted.
- **Error handling**: If `Shutdown()` returns an error (e.g., `context.DeadlineExceeded`), it means not all handlers finished within the allotted time. In this case, you can decide to force termination.

Full graceful shutdown implementations can be found in `examples/simple/main.go` and `examples/typed/main.go`.

## Marshallers

The library ships with two ready-to-use marshallers:

- `marshaller.DefaultJsonMarshaller`
- `marshaller.DefaultProtoMarshaller`

Both implement the `marshaller.PayloadMarshaller` interface. You can replace them with a custom implementation via `HandlerOptions`, `CallOptions`, `EventHandlerOptions`, or `EventPublishOptions`.

## Examples

- `examples/simple` — a basic scenario with JSON handlers that demonstrates RPC, events, and JetStream:

```42:68:examples/simple/router.go
// ... existing code ...
```

- `examples/typed` — typed RPC and event handlers with generics for JSON payloads:

```12:38:examples/typed/router.go
// ... existing code ...
```

Run the examples directly (`go run ./examples/simple`, `go run ./examples/typed`) after spinning up a local NATS (`docker-compose up nats` or `nats-server`).

## Local Development

1. Install dependencies: `go mod tidy`.
2. Start NATS (see `docker-compose.yml`).
3. Run integration tests: `go test ./...`.

Contributions and feature ideas are welcome!
