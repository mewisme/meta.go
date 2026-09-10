# @meewmeew/meta

Node.js and TypeScript client for the `meta.go` gRPC runtime.

```ts
import {MetaClient} from "@meewmeew/meta"

const meta = await MetaClient.create({mode: "managed", runtimePath: "/path/to/meta-runtime"})
const session = await meta.createSession({cookies: {c_user: "...", xs: "..."}})

const events = meta.events(session.sessionId)
events.on("message", event => console.log(event.payload.value.message))

for await (const event of events) console.log(event.payload?.$case)

const snapshot = await meta.refreshAuth(session.sessionId, {appState: [
  {key: "c_user", value: "..."},
  {key: "xs", value: "..."},
]})
```

Sessions accept exactly one auth source: `{cookies}`, `{appState}`, or `{credentials}`. Credentials require `identifier`, `password`, and exactly one of `totp` or `otp`. `refreshAuth()` rotates auth in-place without replacing the runtime session or SDK event streams; `authSnapshot()` exports normalized cookies, AppState and Facebook session metadata.

Use `{mode: "external", endpoint, token}` to connect to an already running runtime. Managed mode resolves `runtimePath`, then `META_RUNTIME_PATH`, then `meta-runtime` on `PATH`. If none is available it downloads the runtime release matching the SDK package version, verifies it against the release SHA-256 manifest and caches it per platform.

Override selection with `runtimeVersion` / `META_RUNTIME_VERSION`, the cache with `runtimeCacheDir` / `META_RUNTIME_CACHE_DIR`, or disable downloading with `downloadRuntime: false`. `releaseBaseUrl` / `META_RUNTIME_RELEASE_BASE_URL` is available for trusted mirrors and tests. An explicitly installed runtime always remains supported; the protocol-major handshake still rejects incompatible binaries after launch.
