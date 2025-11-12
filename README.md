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

eventsSvc := events.NewEvents(nc)
rpcSvc := rpc.NewNatsRPC(nc)

rpcSvc.AddRPCHandler("user.ping", func(ctx rpc.RPCContext) error {
    var req PingRequest
    if err := ctx.Request(&req); err != nil {
        return err
    }
    return ctx.Ok(PingResponse{Message: "pong"})
}, nil)

eventsSvc.AddEventHandler("user.created", func(ctx events.EventContext) error {
    var payload UserCreatedEvent
    if err := ctx.Event(&payload); err != nil {
        return err
    }
    fmt.Printf("created: %#v\n", payload)
    return nil
}, nil)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go rpcSvc.StartWithContext(ctx)
go eventsSvc.StartWithContext(ctx)

_ = eventsSvc.Emit(ctx, "user.created", UserCreatedEvent{ID: "42"}, nil)
var resp PingResponse
_ = rpcSvc.CallRPC("user.ping", PingRequest{ID: "42"}, &resp, rpc.CallOptions{})
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

## Events

- `events.NewEvents` creates a service that supports standard subscriptions and JetStream.
- `AddEventHandler` lets you specify a queue (`Queue`) and JetStream options: durable, pull, deliver group, subject transform, etc.
- For pull consumers, set `JetStream.Pull = true`, `PullBatch`, `PullExpire`, and `Durable`.
- `EventContext` provides:
  - `Event(&payload)` — message deserialization;
  - `Ack/Nak/Term/InProgress` — JetStream delivery control;
  - access to `Headers()` and the original `*nats.Msg`.
- Emit events with `Emit(ctx, subject, payload, opts)`; you can include headers and `nats.PubOpt`.
- Typed helpers: `events.AddTypedEventHandler`, `AddTypedJsonEventHandler`, `AddTypedProtoEventHandler`.

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
