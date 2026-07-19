# deez-nats-gen (`dnatsgen`)

Code generator that turns a **protobuf service definition** into ready-to-use
[deez-nats](..) clients and servers — for **RPC** (request/reply) and
**events** (core pub/sub + JetStream) — in **Go, Kotlin, and Rust**.

One `.proto` is the single source of truth. Today the same NATS contract is
hand-mirrored in the Go service (`pkg/rpc/*` + handlers/clients) and in each
Kotlin mod (`net/nats/Subjects.kt` + `NatsManager.kt` + `dto/*`); drift is silent
and only bites at runtime. `dnatsgen` removes the hand-mirroring.

- **Wire format:** protobuf binary, over the deez-nats `DefaultProtoMarshaller`
  envelope `ProtobufWrap { google.protobuf.Any data = 1; string error = 2; }`.
  Go and Kotlin are **byte-identical** on the wire (proven by a golden test).
- **Engine:** pure Go ([`bufbuild/protocompile`](https://github.com/bufbuild/protocompile)
  parses `.proto` in-process). No `protoc`, no `buf`. Message Go types come from
  the official `protoc-gen-go`, which the generator drives directly.

## The contract: `deeznats.*` options

A contract proto imports the (embedded) `deeznats/annotations.proto` and annotates
its services/methods:

| Option | On | Meaning |
|---|---|---|
| `subject_prefix` | service | prefix joined to each method subject (`"economy"`) |
| `events` | service | whole service is fire-and-forget events, not RPC |
| `jetstream` | service | events transport: `true` = JetStream, `false` = core |
| `stream` | service | JetStream stream name (when `jetstream`) |
| `subject` | method | subject leaf (`"wallet.get"` → `economy.wallet.get`); defaults to `GetBalance → get.balance` |

```proto
syntax = "proto3";
package economy;
import "deeznats/annotations.proto";
import "google/protobuf/empty.proto";
option go_package = ".../gen/go/economypb;economypb";

service Wallet {                                   // RPC service
  option (deeznats.subject_prefix) = "economy";
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse) {
    option (deeznats.subject) = "wallet.get";      // -> economy.wallet.get
  }
}

service Audit {                                    // JetStream event service
  option (deeznats.subject_prefix) = "audit";
  option (deeznats.events) = true;
  option (deeznats.jetstream) = true;
  option (deeznats.stream) = "AUDIT";
  rpc Logged(AuditEntry) returns (google.protobuf.Empty) {
    option (deeznats.subject) = "logged";          // -> audit.logged
  }
}
```

See [`examples/contract/contract.proto`](examples/contract/contract.proto) for a
full example exercising RPC + core events + JetStream events.

## Running the generator

```sh
# Go (messages via protoc-gen-go + NATS glue)
go run ./cmd/dnatsgen \
  -I path/to/protos \
  -proto economy.proto \
  -go-out backend/goeconomy/internal/gen/economypb

# Kotlin (@Serializable messages + client/event glue + DnatsProto runtime)
go run ./cmd/dnatsgen \
  -I path/to/protos \
  -proto economy.proto \
  -kt-out mods/spaceeconomy-mod/src/main/kotlin/ru/lnik801l/economy/gen \
  -kt-package ru.lnik801l.economy

# Rust (prost messages + async client/server/event glue + dnats runtime)
go run ./cmd/dnatsgen \
  -I path/to/protos \
  -proto economy.proto \
  -rust-out crates/economy/src
```

Targets combine in one invocation (`-go-out`/`-kt-out`/`-rust-out` are
independent); each is skipped when its flag is empty.

`dnatsgen` resolves `protoc-gen-go` by running it from the target Go module via
`go run google.golang.org/protobuf/cmd/protoc-gen-go` (override with the
`DNATSGEN_PROTOC_GEN_GO` env var). The `deeznats/annotations.proto` import is
served from the generator's embedded copy — it need not exist on disk.

## What it generates

**Go** (one package, same as the messages):
- `*.pb.go` — protobuf message types (`protoc-gen-go`).
- `<stem>_dnats.gen.go` — subjects consts; per RPC service a `…Client` (proto
  call marshaller) + `…Server` interface + `Register…Server`; per event service a
  `…Publisher` + `…Handler` interface + `Register…` (core or JetStream).
  Clients can wrap an existing `natsrpc.NatsRPC`; publishers can wrap an
  existing core/JetStream events router. Generated RPC/event methods accept
  call/emit options (including headers) and always force the protobuf
  marshaller after caller options.

**Kotlin**:
- `Messages.kt` — `@Serializable` data classes / enums with `@ProtoNumber` /
  `@ProtoType`, encoded by `kotlinx-serialization-protobuf`.
- `Api.kt` — `Subjects`, typed `…Client` (blocking RPC), `…Publisher`, `…Subscriber`.
- `runtime/DnatsProto.kt` — the envelope runtime (emit once per Kotlin module;
  `-kt-runtime=false` to skip).

Kotlin needs `org.jetbrains.kotlinx:kotlinx-serialization-protobuf` + `io.nats:jnats`
and the `kotlin("plugin.serialization")` plugin (already applied in every mod).

**Rust** (one module per proto + a shared runtime):
- `<stem>.rs` — `#[derive(prost::Message)]` structs / `prost::Enumeration` enums
  (byte-compatible with Go/Kotlin), `subjects` consts, per RPC service an async
  `…Client` + `…Server` trait + `serve_…`, per event service a `…Publisher` +
  `…Handler` trait + `subscribe_…` (core or JetStream).
- `dnats.rs` — the envelope runtime (emit once per crate; `-rust-runtime=false`
  to skip). Redirect it with `-rust-runtime-out`.

Rust needs `prost`, `async-nats`, `async-trait`, `futures`, and a `tokio`
runtime; the generated modules are wired into a crate by a hand-written
`lib.rs` (`pub mod dnats; pub mod <stem>;`) — see `examples/contract/rust`.
References to messages/enums from another proto are emitted as imports from the
generated sibling module (for example, `events.rs` imports types from
`crate::rpc`). JetStream service stream names are exported next to subject
constants as `<SERVICE>_STREAM`.

In the digiversity monorepo all owner contracts and their external Kotlin/Rust
consumers are generated from the repository root:

```sh
scripts/generate-dnats.sh
scripts/generate-dnats.sh --check
```

## Verification

```sh
# Go: builds + vets + in-process embedded-NATS e2e (RPC, core, JetStream) + golden wire
go build ./... && go vet ./... && go test ./...

# Kotlin: compiles the generated code against Kotlin 2.0.0 + serialization + jnats
cd examples/contract/kotlin && ./gradlew compileKotlin

# Cross-language: Kotlin decodes Go's exact wire bytes and re-encodes byte-identical
cd examples/contract/kotlin && ./gradlew run

# Rust: compiles the generated crate + decodes Go's exact wire bytes byte-identical
cd examples/contract/rust && cargo test
```

`cmd/xlangdump` prints the Go wire bytes used to seed the cross-language golden.
The same `goldenEnvelopeHex` is asserted byte-identical by Go, Kotlin, and Rust.

## v1 scope / limitations

- `oneof` is **rejected with a generation error** (it has no faithful Kotlin
  mapping yet — split it into separate fields). `proto3 optional` is a *synthetic*
  oneof and is allowed (its presence is not yet carried into Kotlin).
- **Streaming** methods are **skipped with a console warning** — deez-nats RPC is
  unary request/reply. Event methods must `returns (google.protobuf.Empty)`.
- An event service is uniform: one transport, one stream (one deez-nats router).
- Kotlin map key/value integers use the default varint encoding (no per-entry
  `@ProtoType`); strings and the common scalar types are fully supported.

Graceful shutdown with multiple RPC/event subscriptions is covered by the Go
e2e and tracker regression tests.
