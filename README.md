# fbgo

`fbgo` is a Go library and CLI for Facebook Messenger automation, including regular Messenger and E2EE workflows.

Module:

```text
go.mewis.me/fbgo
```

The root package is library-first. The `fbgo` binary is a thin CLI over the same capabilities exposed to Go callers.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
govulncheck ./...
```

Third-party dependency license metadata is tracked in `THIRD_PARTY_LICENSES.md`.