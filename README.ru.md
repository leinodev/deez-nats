# deez-nats

Утилиты для построения RPC и event-driven приложений поверх [NATS](https://nats.io) на Go. Библиотека объединяет единый роутер для RPC-методов и событий, поддержку middlewares, типизированные обёртки и несколько маршаллизаторов (JSON и Protobuf), чтобы ускорить старт и облегчить поддержку сервиса.

## Основные возможности

- **Единый роутер** с возможностью группировки (`Group`) и наследованием middleware для RPC и событий.
- **Обработка RPC-запросов** с автоматическим управлением ack/nak и удобным `RPCContext` для чтения запроса, отправки ответа и работы с заголовками.
- **Event-хэндлеры с JetStream**: очередь / pull-консьюмеры, авто-ack, конфигурация durable и subject transform.
- **Типизированные обёртки** для RPC (`rpc.AddTyped…`) и событий (`events.AddTyped…`) с поддержкой generics и выбором маршаллизатора.
- **Гибкие маршаллизаторы**: готовые JSON и Protobuf, возможность подменить на свой.
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

eventsSvc := events.NewNatsEvents(nc, nil)
rpcSvc := rpc.NewNatsRPC(nc, nil)

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
_ = rpcSvc.CallRPC(ctx, "user.ping", PingRequest{ID: "42"}, &resp, rpc.CallOptions{})
```

## RPC

- `rpc.NewNatsRPC` создаёт сервис с дефолтным JSON-маршаллизатором.
- `AddRPCHandler` иерархически строит дерево маршрутов; можно группировать методы (`Service.Group("user")`).
- `RPCContext` предоставляет:
  - `Request(&reqStruct)` — десериализация запроса;
  - `Ok(response)` — отправка успешного ответа;
  - `Headers()` и `RequestHeaders()` — работа с заголовками.
- `CallRPC` инкапсулирует запрос с таймаутом NATS и десериализацией респондов.
- Для generics используйте `rpc.AddTypedJsonRPCHandler`, `AddTypedProtoRPCHandler` или `AddTypedRPCHandler` с кастомным маршаллизатором.

### Билдер опций RPC

Используйте `rpc.NewRPCOptionsBuilder()` для настройки опций RPC-сервиса:

```go
opts := rpc.NewRPCOptionsBuilder().
    WithBaseRoute("myservice").
    WithDefaultHandlerOptions(func(b *rpc.HandlerOptionsBuilder) {
        b.WithMarshaller(customMarshaller)
    }).
    WithDefaultCallOptions(func(b *rpc.CallOptionsBuilder) {
        b.WithHeader("X-Service", "my-service")
    }).
    Build()

rpcSvc := rpc.NewNatsRPC(nc, &opts)
```

## События

- `events.NewNatsEvents` создаёт сервис, поддерживающий обычные подписки и JetStream.
- `AddEventHandler` позволяет указать очередь (`Queue`) и JetStream-настройки: durable, pull, deliver group, subject transform и т.д.
- Для pull-консьюмеров задайте `JetStream.Pull = true`, `PullBatch`, `PullExpire` и `Durable`.
- `EventContext` предоставляет:
  - `Event(&payload)` — десериализация сообщения;
  - `Ack/Nak/Term/InProgress` — управление delivery в JetStream;
  - доступ к `Headers()` и исходному `*nats.Msg`.
- Эмиссия событий через `Emit(ctx, subject, payload, opts)`; можно передать заголовки и `nats.PubOpt`.
- Типизированные обёртки: `events.AddTypedEventHandler`, `AddTypedJsonEventHandler`, `AddTypedProtoEventHandler`.

### Билдер опций Events

Используйте `events.NewEventsOptionsBuilder()` для настройки опций сервиса событий:

```go
opts := events.NewEventsOptionsBuilder().
    WithJetStream(js).
    WithDefaultHandlerOptions(func(b *events.EventHandlerOptionsBuilder) {
        b.WithQueue("my-queue").
          WithJetStream(func(jsb *events.JetStreamEventOptionsBuilder) {
              jsb.Enabled().
                  WithAutoAck(true).
                  WithDurable("my-consumer")
          })
    }).
    WithDefaultPublishOptions(func(b *events.EventPublishOptionsBuilder) {
        b.WithMarshaller(customMarshaller).
          WithHeader("X-Source", "my-service")
    }).
    Build()

eventsSvc := events.NewNatsEvents(nc, &opts)
```

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

    eventService := events.NewNatsEvents(nc, nil)
    rpcService := rpc.NewNatsRPC(nc, nil)

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

## Маршаллизаторы

Библиотека поставляет два готовых маршаллизатора:

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

