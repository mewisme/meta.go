# @meewmeew/meta

Node.js and TypeScript client for the `meta.go` gRPC runtime.

```ts
import {MetaClient} from "@meewmeew/meta"

const meta = await MetaClient.create({mode: "managed", runtimePath: "/path/to/meta-runtime"})
const session = await meta.createSession({c_user: "...", xs: "..."})

const events = meta.events(session.sessionId)
events.on("message", event => console.log(event.payload.value.message))

for await (const event of events) console.log(event.payload?.$case)
```

Use `{mode: "external", endpoint, token}` to connect to an already running runtime. Managed mode resolves the runtime from `runtimePath`, `META_RUNTIME_PATH`, or `meta-runtime` on `PATH`. Runtime downloading is intentionally left to the distribution layer.
