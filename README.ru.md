# deez-nats

<img src="./logo.png" alt="logo" width="250"/>

Утилиты для построения RPC и event-driven приложений поверх [NATS](https://nats.io) на Go. Библиотека объединяет единый роутер для RPC-методов и событий, поддержку middlewares, типизированные обёртки и несколько маршаллеров (JSON и Protobuf), чтобы ускорить старт и облегчить поддержку сервиса.

## Основные возможности

- **Единый роутер** с возможностью группировки (`Group`) и наследованием middleware для RPC и событий.
- **Обработка RPC-запросов** с автоматическим управлением ack/nak и удобным `RPCContext` для чтения запроса, отправки ответа и работы с заголовками.
- **Event-хэндлеры с JetStream**: очередь / pull-консьюмеры, авто-ack, конфигурация durable и subject transform.
- **Типизированные обёртки** для RPC (`rpc.AddTyped…`) и событий (`events.AddTypedCore…` / `events.AddTypedJetStream…`) с поддержкой generics и выбором маршаллера.
- **Гибкие маршаллеры**: готовые JSON и Protobuf, возможность подменить на свой.
- **Примеры** для быстрого старта: простой сценарий и полностью типизированный пайплайн.

## Установка

```sh
go get github.com/leinodev/deez-nats
```

Требуется Go 1.21+ (модуль собирается с `go 1.25`).

## Быстрый старт

```go
nc, _ := nats.Connect(nats.DefaultURL)
defer nc.Close()

eventsSvc := events.NewCoreEvents(nc)
rpcSvc := rpc.NewNatsRPC(nc)

rpcSvc.AddRPCHandler("user.ping", func(ctx rpc.RPCContext) error {
    var req PingRequest
    if err := ctx.Request(&req); err != nil {
        return err
    }
    return ctx.Ok(PingResponse{Message: "pong"})
})

eventsSvc.AddEventHandler("user.created", func(ctx events.EventContext[*nats.Msg, nats.AckOpt]) error {
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

- `rpc.NewNatsRPC` создаёт сервис с дефолтным JSON-маршаллером.
- `AddRPCHandler` иерархически строит дерево маршрутов; можно группировать методы (`Service.Group("user")`).
- `RPCContext` предоставляет:
  - `Request(&reqStruct)` — десериализация запроса;
  - `Ok(response)` — отправка успешного ответа;
  - `Headers()` и `RequestHeaders()` — работа с заголовками.
- `CallRPC` инкапсулирует запрос с таймаутом NATS и десериализацией респондов.
- Для generics используйте `rpc.AddTypedJsonRPCHandler`, `AddTypedProtoRPCHandler` или `AddTypedRPCHandler` с кастомным маршаллером.
- Используйте `rpc.WithHandlerMiddlewares(...)` для добавления middlewares к конкретным обработчикам.

### Опции RPC

Используйте функциональные опции для настройки RPC-сервиса:

```go
rpcSvc := rpc.NewNatsRPC(nc,
    rpc.WithBaseRoute("myservice"),
    rpc.WithDefaultHandlerMarshaller(customMarshaller),
    rpc.WithDefaultCallOptions(
        rpc.WithCallHeader("X-Service", "my-service"),
    ),
)

// Добавить обработчик с middlewares
rpcSvc.AddRPCHandler("user.get", handler,
    rpc.WithHandlerMarshaller(customMarshaller),
    rpc.WithHandlerMiddlewares(
        authMiddleware,
        loggingMiddleware,
    ),
)

// Вызов RPC с опциями
rpcSvc.CallRPC(ctx, "user.ping", request, &response,
    rpc.WithCallHeader("X-Request-ID", "123"),
    rpc.WithCallMarshaller(customMarshaller),
)
```

## События

Библиотека предоставляет две реализации событий:

- **`events.NewCoreEvents`** — стандартные подписки NATS с группами очередей
- **`events.NewJetStreamEvents`** — события на основе JetStream с конфигурацией консьюмеров

### Core Events

```go
nc, _ := nats.Connect(nats.DefaultURL)
defer nc.Close()

coreEvents := events.NewCoreEvents(nc,
    events.WithCoreQueueGroup("my-queue-group"),
    events.WithCoreDefaultEmitMarshaller(customMarshaller),
    events.WithCoreDefaultEmitHeader("X-Service", "my-service"),
    events.WithCoreDefaultEventHandlerMarshaller(customMarshaller),
)

coreEvents.AddEventHandler("user.created", func(ctx events.EventContext[*nats.Msg, nats.AckOpt]) error {
    var payload UserCreatedEvent
    if err := ctx.Event(&payload); err != nil {
        return err
    }
    fmt.Printf("created: %#v\n", payload)
    return nil
}, events.WithCoreHandlerQueue("handler-queue"))

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go coreEvents.StartWithContext(ctx)

_ = coreEvents.Emit(ctx, "user.created", UserCreatedEvent{ID: "42"},
    events.WithCoreEmitHeader("X-Request-ID", "123"),
)
```

### JetStream Events

```go
js, _ := jetstream.New(nc)

jetStreamEvents := events.NewJetStreamEvents(js,
    events.WithJetStreamStream("EVENTS"),
    events.WithJetStreamDeliverGroup("events-group"),
    events.WithJetStreamDefaultEmitMarshaller(customMarshaller),
    events.WithJetStreamDefaultEmitHeader("X-Service", "my-service"),
    events.WithJetStreamDefaultEventHandlerMarshaller(customMarshaller),
)

jetStreamEvents.AddEventHandler("entity.created", handler,
    events.WithJetStreamHandlerMarshaller(customMarshaller),
)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go jetStreamEvents.StartWithContext(ctx)

_ = jetStreamEvents.Emit(ctx, "entity.created", payload,
    events.WithJetStreamEmitMarshaller(customMarshaller),
    events.WithJetStreamEmitHeader("X-Request-ID", "123"),
)
```

### Опции Events

**Core Events Options:**
- `WithCoreQueueGroup(queueGroup)` — группа очереди по умолчанию для всех обработчиков
- `WithCoreDefaultEmitMarshaller(m)` — маршаллер по умолчанию для всех эмиссий
- `WithCoreDefaultEmitHeader(key, value)` — заголовок по умолчанию для всех эмиссий
- `WithCoreDefaultEmitHeaders(headers)` — заголовки по умолчанию для всех эмиссий
- `WithCoreDefaultEventHandlerMarshaller(m)` — маршаллер по умолчанию для всех обработчиков
- `WithCoreHandlerMarshaller(m)` — маршаллер для конкретного обработчика
- `WithCoreHandlerQueue(queue)` — очередь для конкретного обработчика (переопределяет значение по умолчанию)
- `WithCoreEmitMarshaller(m)` — маршаллер для конкретной эмиссии (переопределяет значение по умолчанию)
- `WithCoreEmitHeader(key, value)` — заголовок для конкретной эмиссии (объединяется со значениями по умолчанию)
- `WithCoreEmitHeaders(headers)` — заголовки для конкретной эмиссии (объединяются со значениями по умолчанию)

**JetStream Events Options:**
- `WithJetStreamStream(stream)` — имя потока JetStream
- `WithJetStreamDeliverGroup(group)` — группа доставки для консьюмеров
- `WithJetStreamDefaultEmitMarshaller(m)` — маршаллер по умолчанию для всех эмиссий
- `WithJetStreamDefaultEmitHeader(key, value)` — заголовок по умолчанию для всех эмиссий
- `WithJetStreamDefaultEmitHeaders(headers)` — заголовки по умолчанию для всех эмиссий
- `WithJetStreamDefaultEventHandlerMarshaller(m)` — маршаллер по умолчанию для всех обработчиков
- `WithJetStreamHandlerMarshaller(m)` — маршаллер для конкретного обработчика
- `WithJetStreamEmitMarshaller(m)` — маршаллер для конкретной эмиссии (переопределяет значение по умолчанию)
- `WithJetStreamEmitHeader(key, value)` — заголовок для конкретной эмиссии (объединяется со значениями по умолчанию)

### EventContext

`EventContext` предоставляет:
- `Event(&payload)` — десериализация сообщения
- `Ack/Nak/Term/InProgress` — управление доставкой в JetStream (для JetStream событий)
- `Headers()` — доступ к заголовкам сообщения
- `Message()` — доступ к исходному сообщению

**Примечание:** `DefaultHeaders` в `JetStreamEventHandlerOptions` в настоящее время определены, но не используются активно при обработке обработчиков. Они зарезервированы для будущей функциональности.

### Типизированные обработчики событий

Для типобезопасной обработки событий с использованием generics используйте типизированные хелперы:

**Core Events:**
```go
events.AddTypedCoreJsonEventHandler(coreEvents, "user.created", func(ctx events.EventContext[*nats.Msg, nats.AckOpt], payload UserCreatedEvent) error {
    fmt.Printf("user created: %#v\n", payload)
    return nil
})

events.AddTypedCoreProtoEventHandler(coreEvents, "user.updated", func(ctx events.EventContext[*nats.Msg, nats.AckOpt], payload UserUpdatedEvent) error {
    fmt.Printf("user updated: %#v\n", payload)
    return nil
}, events.WithCoreHandlerQueue("user-queue"))
```

**JetStream Events:**
```go
events.AddTypedJetStreamJsonEventHandler(jetStreamEvents, "entity.created", func(ctx events.EventContext[jetstream.Msg, any], payload EntityCreatedEvent) error {
    fmt.Printf("entity created: %#v\n", payload)
    return nil
})

events.AddTypedJetStreamEventHandlerWithMarshaller(jetStreamEvents, "entity.updated", func(ctx events.EventContext[jetstream.Msg, any], payload EntityUpdatedEvent) error {
    fmt.Printf("entity updated: %#v\n", payload)
    return nil
}, customMarshaller, events.WithJetStreamHandlerMarshaller(customMarshaller))
```

Доступные типизированные хелперы:
- `AddTypedCoreEventHandler` / `AddTypedJetStreamEventHandler` — базовые функции
- `AddTypedCoreEventHandlerWithMarshaller` / `AddTypedJetStreamEventHandlerWithMarshaller` — с кастомным маршаллером
- `AddTypedCoreJsonEventHandler` / `AddTypedJetStreamJsonEventHandler` — с JSON маршаллером
- `AddTypedCoreProtoEventHandler` / `AddTypedJetStreamProtoEventHandler` — с Protobuf маршаллером

## Graceful Shutdown

Библиотека поддерживает graceful shutdown для RPC и событийных сервисов. Это позволяет корректно завершить обработку активных запросов и событий перед остановкой сервиса.

### Использование

Оба сервиса (`rpc.NatsRPC` и `events.NatsEvents`) реализуют метод `Shutdown(ctx context.Context) error`, который:

1. **Останавливает приём новых сообщений** — подписки переводятся в режим drain, новые запросы и события не принимаются.
2. **Ожидает завершения активных обработчиков** — все запущенные обработчики получают возможность завершить работу.
3. **Отписывается от всех подписок** — после завершения всех обработчиков происходит отписка от NATS subjects.

### Пример использования

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

    eventService := events.NewCoreEvents(nc)
    rpcService := rpc.NewNatsRPC(nc)

    // ... настройка обработчиков ...

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go eventService.StartWithContext(ctx)
    go rpcService.StartWithContext(ctx)

    // Ожидание сигнала завершения (SIGINT или SIGTERM)
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    <-sigChan
    log.Println("Received shutdown signal, starting graceful shutdown...")

    // Отменяем контекст, чтобы остановить обработку новых сообщений
    cancel()

    // Выполняем graceful shutdown с таймаутом
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

### Важные моменты

- **Таймаут**: Всегда используйте контекст с таймаутом для `Shutdown()`, чтобы избежать бесконечного ожидания.
- **Порядок**: Сначала отмените контекст (`cancel()`), затем вызовите `Shutdown()` — это гарантирует, что новые сообщения не будут приняты.
- **Обработка ошибок**: Если `Shutdown()` возвращает ошибку (например, `context.DeadlineExceeded`), это означает, что не все обработчики успели завершиться в отведённое время. В этом случае можно принять решение о принудительном завершении.

Примеры с полной реализацией graceful shutdown можно найти в `examples/simple/main.go` и `examples/typed/main.go`.

## маршаллеры

Библиотека поставляет два готовых маршаллера:

- `marshaller.DefaultJsonMarshaller`
- `marshaller.DefaultProtoMarshaller`

Оба реализуют интерфейс `marshaller.PayloadMarshaller`. Его можно заменить на пользовательский и пробросить через `HandlerOptions`, `CallOptions`, `EventHandlerOptions` или `EventPublishOptions`.

## Примеры

- `examples/simple` — базовый сценарий с JSON-хэндлерами, демонстрирующий работу RPC, событий и JetStream:

```42:68:examples/simple/router.go
// ... existing code ...
```

- `examples/typed` — типизированные RPC и event-хэндлеры с generics для JSON-полезных нагрузок:

```12:38:examples/typed/router.go
// ... existing code ...
```

Запустите примеры напрямую (`go run ./examples/simple`, `go run ./examples/typed`), предварительно подняв локальный NATS (`docker-compose up nats` или `nats-server`).

## Локальная разработка

1. Установите зависимости: `go mod tidy`.
2. Поднимите NATS (см. `docker-compose.yml`).
3. Запустите интеграционные тесты: `go test ./...`.

Будем рады issue и предложениям по улучшению!

