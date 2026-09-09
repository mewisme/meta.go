# Contributing

## Development setup

Use the Go version declared in `go.mod`.

```sh
go mod verify
go test ./...
go test -race ./...
go vet ./...
govulncheck ./...
```

Run `gofmt` on changed Go files and keep `go.mod`/`go.sum` tidy.

## Design rules

- Keep the root package library-first.
- Network/blocking public operations accept `context.Context`.
- Prefer typed models and errors over raw maps or protocol-library types.
- Keep Facebook protocol details inside internal adapters/registries.
- Do not leak `mautrix-meta`, `messagix` or `whatsmeow` types through public APIs.
- Keep IDs string-backed at public boundaries.
- Do not silently downgrade operations that require E2EE.
- Preserve bounded event delivery and invoke user handlers outside internal locks.
- Never log cookies, passwords, TOTP seeds, tokens or E2EE key material.

## Protocol changes

When a private endpoint, task, persisted query or response schema changes:

1. reproduce the failure with a sanitized fixture or injected test server;
2. identify whether the change belongs in the protocol registry, parser or transport adapter;
3. add a regression test before or with the fix;
4. preserve the existing typed public behavior unless a versioned API change is necessary;
5. classify remote protocol-shape failures as `ErrProtocolChanged` rather than returning raw response bodies;
6. run unit, race and vulnerability gates.

Do not spread unstable document IDs, task labels or endpoint strings through public service packages.

## Security-sensitive changes

Changes involving authentication, secret storage, E2EE state, redirects, file writes or media fetching require explicit negative tests for malformed/hostile inputs.

## Release-sensitive changes

If direct dependencies change, update `THIRD_PARTY_LICENSES.md`. Changes to build targets, ldflags, artifact contents or release metadata must also update `.goreleaser.yaml`, release documentation and relevant verification tests.
