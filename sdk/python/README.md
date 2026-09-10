# meewmeew-meta

Async Python SDK for the `meta.go` gRPC runtime.

```python
from meewmeew_meta import MetaClient

async with await MetaClient.managed(runtime_path="/path/to/meta-runtime") as meta:
    session = await meta.create_session({"c_user": "...", "xs": "..."})
    events = meta.events(session.session_id)
    events.on("message", lambda event: print(event.message.message))

    async for event in events:
        print(event.WhichOneof("payload"))
```

Use `await MetaClient.external(endpoint, token)` to connect to an already running runtime. Managed mode resolves the runtime from `runtime_path`, `META_RUNTIME_PATH`, or `meta-runtime` on `PATH`. Runtime downloading is intentionally left to the distribution phase.
