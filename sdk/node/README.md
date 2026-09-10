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

Use `{mode: "external", endpoint, token}` to connect to an already running runtime. Managed mode resolves `runtimePath`, then `META_RUNTIME_PATH`, then the matching bundled `@meewmeew/meta-runtime-*` optional package, then `meta-runtime` on `PATH`. If none is available it fails locally without downloading anything.

Bundled runtime packages are version-pinned to the root SDK during release preparation, so the default managed runtime comes from the same release. An explicitly installed runtime remains supported as an override; the protocol-major handshake still rejects incompatible binaries after launch.
