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

- `rpc.NewNatsRPC` создаёт сервис с дефолтным JSON-маршаллизатором.
- `AddRPCHandler` иерархически строит дерево маршрутов; можно группировать методы (`Service.Group("user")`).
- `RPCContext` предоставляет:
  - `Request(&reqStruct)` — десериализация запроса;
  - `Ok(response)` — отправка успешного ответа;
  - `Headers()` и `RequestHeaders()` — работа с заголовками.
- `CallRPC` инкапсулирует запрос с таймаутом NATS и десериализацией респондов.
- Для generics используйте `rpc.AddTypedJsonRPCHandler`, `AddTypedProtoRPCHandler` или `AddTypedRPCHandler` с кастомным маршаллизатором.

## События

- `events.NewEvents` создаёт сервис, поддерживающий обычные подписки и JetStream.
- `AddEventHandler` позволяет указать очередь (`Queue`) и JetStream-настройки: durable, pull, deliver group, subject transform и т.д.
- Для pull-консьюмеров задайте `JetStream.Pull = true`, `PullBatch`, `PullExpire` и `Durable`.
- `EventContext` предоставляет:
  - `Event(&payload)` — десериализация сообщения;
  - `Ack/Nak/Term/InProgress` — управление delivery в JetStream;
  - доступ к `Headers()` и исходному `*nats.Msg`.
- Эмиссия событий через `Emit(ctx, subject, payload, opts)`; можно передать заголовки и `nats.PubOpt`.
- Типизированные обёртки: `events.AddTypedEventHandler`, `AddTypedJsonEventHandler`, `AddTypedProtoEventHandler`.

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

