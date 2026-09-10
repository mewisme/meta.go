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

Use `await MetaClient.external(endpoint, token)` to connect to an already running runtime. Managed mode resolves `runtime_path`, then `META_RUNTIME_PATH`, then `meta-runtime` on `PATH`. If none is available it downloads the runtime release matching the SDK package version, verifies it against the release SHA-256 manifest and caches it per platform.

Override selection with `runtime_version` / `META_RUNTIME_VERSION`, the cache with `runtime_cache_dir` / `META_RUNTIME_CACHE_DIR`, or disable downloading with `download_runtime=False`. `release_base_url` / `META_RUNTIME_RELEASE_BASE_URL` is available for trusted mirrors and tests. An explicitly installed runtime always remains supported; the protocol-major handshake still rejects incompatible binaries after launch.
