# Multi-language runtime architecture

`meta.go` keeps all Facebook, Messenger, E2EE and Lightspeed behavior in Go. The multi-language runtime is an additional RPC boundary for SDKs; it does not replace the in-process Go API.

## Boundary

```text
SDK -> protobuf/gRPC -> runtime adapter -> meta.go Client/services -> engine/backend
```

Protobuf types stop at `internal/runtime`. They must not leak into `model`, `messenger`, `thread`, `facebook`, or `internal/meta`.

The runtime hosts multiple isolated sessions in one process. Each session owns one `meta.Client`, its connection lifecycle, account state, event fan-out, health state and cleanup. Disconnecting an event subscriber does not destroy the session.

## RPC domains

### Runtime

- runtime/protocol version and build metadata
- capability discovery
- server health metadata that is not session-specific

### Session

- create session from supplied auth/session material
- connect session
- close session
- get account and health state
- subscribe to normalized events

### Messenger

Regular Messenger and thread operations:

- send, forward, share contact
- react/remove reaction, edit, unsend, typing, read
- message requests
- theme list/find/set
- note get/create/delete/recreate
- Messenger restrict and message-block state changes
- thread list/get/create DM/delete
- poll create/vote/details
- pinned-message list and pin/unpin
- message search
- thread mute/call mute/approval/archive/admin/name/emoji/nickname
- Messenger user search/contact lookup

Regular media upload and thread-photo upload are streaming operations and are introduced with the media-streaming contract instead of an unbounded unary `bytes` field.

### Facebook

- user lookup and search
- notifications
- bio update
- additional profile creation
- unfriend/block
- post create/archive/delete
- Marketplace listing create/get
- Professional Mode update

### E2EE

- text send
- react/edit/unsend/typing/read

E2EE media send/download are streaming operations and are introduced with the media-streaming contract.

## Event inventory

One `SessionService.SubscribeEvents` server stream carries every normalized event kind:

- `ready`
- `reconnected`
- `disconnected`
- `error`
- `message`
- `messageEdit`
- `messageUnsend`
- `reaction`
- `typing`
- `readReceipt`
- `deliveryReceipt`
- `threadUpdate`
- `threadSystem`
- `e2eeReady`
- `e2eeReceipt`

The event envelope adds runtime-local `session_id`, monotonically increasing per-session `sequence`, and `emitted_at`. Delivery is at-most-once in protocol v1. Slow subscribers use bounded queues; they cannot block engine event production indefinitely. Subscriber drops are tracked independently from engine drops.

## Contract conventions

- Protobuf package: `meta.v1`.
- Go generated package: `go.mewis.me/meta.go/gen/go/meta/v1`.
- RPC/message names use PascalCase; fields use protobuf snake_case.
- Facebook/Messenger IDs cross the RPC boundary as strings.
- Absolute times use `google.protobuf.Timestamp`.
- Durations use `google.protobuf.Duration`.
- Optional fields use message presence or `optional` when absence differs from zero/empty.
- Small bounded binary values may use `bytes`; media payloads use streaming.
- Removed protobuf field numbers are reserved and never reused.
- Additive fields, enum values, RPCs and event variants are compatible within `meta.v1`.
- Breaking semantic/schema changes require `meta.v2`.

## Error contract

Runtime errors use normal gRPC status codes plus stable machine-readable metadata derived from `errors.Classify`.

Stable categories:

- `unknown`
- `auth`
- `checkpoint`
- `rate_limit`
- `invalid_input`
- `connection`
- `e2ee`
- `protocol`
- `permission`
- `network`
- `canceled`

The detail model contains category, code, message, retryable and optional key/value details. SDKs must not depend on arbitrary Go error strings.

## Capability contract

`RuntimeService.GetInfo` reports runtime version, RPC protocol major/minor, build/platform metadata and capability strings. New SDKs must check a capability before using functionality that may not exist in older v1 runtimes.

Capability names use stable dotted lowercase identifiers grouped by domain, for example:

```text
session.events
messenger.send
messenger.threads
messenger.search
facebook.search
e2ee.send
media.streaming
```

A capability is added only when availability may differ between runtime versions/builds; it is not a second schema registry.

## Toolchain

The protobuf toolchain is pinned by repository configuration/scripts:

- Buf CLI `v1.72.0`
- `protoc-gen-go` from `google.golang.org/protobuf v1.36.12`
- `protoc-gen-go-grpc v1.6.2`
- `google.golang.org/grpc v1.83.2`

Generation must be reproducible without requiring globally installed tools. CI lints schemas, checks breaking changes where a comparison baseline is available, regenerates bindings and fails on generated drift.

## Runtime distribution

GoReleaser publishes `meta-runtime` as a standalone executable for Linux, macOS and Windows on amd64 and arm64. Runtime executables are included in the same SHA-256 checksum manifest and provenance flow as the normal CLI release artifacts.

Node and Python use verified lazy download instead of bundling every platform binary into each SDK package. Managed runtime selection is deterministic:

1. explicit SDK runtime path;
2. `META_RUNTIME_PATH`;
3. an installed `meta-runtime` on `PATH`;
4. the runtime release matching the SDK package version, or an explicit runtime-version override.

Downloaded executables are selected from the host OS/architecture, verified against the release checksum manifest before execution and cached by runtime version plus target. SDK package and runtime release versions may advance independently at the protocol level, but the default managed download pins the same release version for reproducibility. The runtime protocol-major handshake and capability checks remain the final compatibility boundary. External runtime mode never downloads or launches a process.

## Compatibility policy

Runtime release versions and RPC protocol versions are independent.

- Old v1 SDK + newer v1 runtime must keep working for existing fields/RPCs.
- Newer v1 SDK + older v1 runtime must fail clearly when a required capability is missing.
- Unknown additive protobuf fields are ignored/preserved according to each generated protobuf implementation.
- A v1 runtime may add RPCs and event variants without changing the protocol major.
- Incompatible changes require `meta.v2`; a migration release may serve v1 and v2 together.

## Future feature workflow

A normal new cross-language method requires:

```text
engine/model change
-> protobuf declaration
-> one Go runtime adapter
-> regenerate clients
-> optional thin SDK ergonomics
-> tests
```

A new event requires:

```text
normalized model.Event
-> protobuf event variant
-> one conversion mapping
-> regenerate clients
-> optional SDK event-name helper
-> tests
```

No feature may introduce a language-specific native bridge, SDK-specific protocol implementation, separate event transport, or Go ABI/FFI dependency.
