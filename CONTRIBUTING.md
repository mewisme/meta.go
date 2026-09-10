# Contributing

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

Report security vulnerabilities privately per [SECURITY.md](SECURITY.md). Do not open public issues for security bugs or attach cookies, passwords, TOTP seeds, tokens or E2EE key material.

Use the [bug](.github/ISSUE_TEMPLATE/bug_report.yml) and [feature](.github/ISSUE_TEMPLATE/feature_request.yml) issue templates when filing issues. Pull requests should follow the [PR template](.github/PULL_REQUEST_TEMPLATE.md).

## Development setup

Use the Go version declared in `go.mod`.

A local `go.work` that includes `../meta-extra` is optional for development and is gitignored. CI and release gates run with `GOWORK=off`; verify changes against the module as published, not only against a workspace replace.

```sh
go mod verify
test -z "$(gofmt -l .)"
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go test ./...
go test -race ./...
go mod tidy && git diff --exit-code -- go.mod go.sum
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

These commands match the primary CI job in [`.github/workflows/ci.yml`](.github/workflows/ci.yml). Run `gofmt` on changed Go files and keep `go.mod`/`go.sum` tidy.

Optional: install [pre-commit](https://pre-commit.com/) and run `pre-commit install` to enable the hooks in [`.pre-commit-config.yaml`](.pre-commit-config.yaml).

## Pull requests

- Keep changes focused; prefer small PRs over mixed refactors.
- Add or update tests for behavior changes.
- Pass the local gates above before requesting review.
- Do not commit secrets, live cookies or private E2EE state.
- If direct dependencies change, update [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md).
- If build targets, ldflags or release artifacts change, update [`.goreleaser.yaml`](.goreleaser.yaml) and [RELEASING.md](RELEASING.md).

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

## Cross-language runtime changes

The initial SDK scope is Node.js/TypeScript and Python. Do not add another language SDK just because a new runtime feature is introduced.

For a normal feature that must be available across languages:

1. implement the engine/model behavior in Go and keep protocol-specific details behind existing adapters;
2. add the protobuf request/response types and RPC under `proto/meta/v1`, using only additive v1 changes and never reusing removed field numbers;
3. add exactly one Go runtime adapter that maps protobuf types to the existing Go API;
4. add a capability string only when availability can differ between runtime versions/builds;
5. regenerate Go, Node and Python bindings with `./scripts/proto.sh generate`, `./scripts/proto.sh generate-node` and `./scripts/proto.sh generate-python`;
6. add thin Node/Python convenience APIs only when generated stubs are not ergonomic enough;
7. add Go runtime tests plus matching Node/Python conformance coverage;
8. run `./scripts/proto.sh check`, `./scripts/proto.sh check-node`, `./scripts/proto.sh check-python` and `./scripts/proto.sh check-future` before committing.

For a new event, add the normalized Go event first, add one protobuf `Event.payload` variant with a new field number, update the single runtime conversion mapping, regenerate Node/Python bindings and add stream tests. Older SDKs must continue to tolerate unknown additive fields: generic event iteration keeps the known envelope, while typed listeners only receive variants known to that SDK. Python protobuf currently preserves unknown wire fields when re-serializing; the Node generator safely ignores them.

`testdata/future-v1` is a synthetic compatibility fixture. `./scripts/proto.sh check-future` copies the current v1 schema, injects a future `RuntimeService.FutureEcho` RPC and `Event.future` variant, then runs the same Node/Python generators in a temporary directory. Keep this fixture synthetic; never expose `future.echo` as a real capability unless an actual feature with that contract is implemented.

## Security-sensitive changes

Changes involving authentication, secret storage, E2EE state, redirects, file writes or media fetching require explicit negative tests for malformed/hostile inputs.

## Release-sensitive changes

If direct dependencies change, update `THIRD_PARTY_LICENSES.md`. Changes to build targets, ldflags, artifact contents or release metadata must also update `.goreleaser.yaml`, release documentation and relevant verification tests.
