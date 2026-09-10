# meta.go

[![CI](https://github.com/mewisme/meta.go/actions/workflows/ci.yml/badge.svg)](https://github.com/mewisme/meta.go/actions/workflows/ci.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/github/go-mod/go-version/mewisme/meta.go)](go.mod)
[![Release](https://img.shields.io/github/v/release/mewisme/meta.go?display_name=tag)](https://github.com/mewisme/meta.go/releases)

Go-first Facebook Messenger and Facebook automation library with a `meta` CLI. Supports regular Messenger and end-to-end encrypted messaging. Pure Go at runtime — no Python or protocol subprocess.

Canonical module path: `go.mewis.me/meta.go`

## Install

Library (after a tagged release):

```sh
go get go.mewis.me/meta.go@latest
```

CLI from source:

```sh
git clone https://github.com/mewisme/meta.go.git
cd meta.go
go install ./cmd/meta
```

Full install options: [Installation](docs/installation.md).

## Library quick start

```go
client, err := meta.NewClient(
	meta.WithCookies(auth.Cookies{"c_user": "...", "xs": "..."}),
	meta.WithE2EE(true),
)
if err != nil {
	return err
}
defer client.Close()

if err := client.Connect(ctx); err != nil {
	return err
}

result, err := client.Messenger.Send(ctx, meta.SendRequest{
	ThreadID: "1234567890",
	Text:     "hello from meta",
})
```

Full example: [Library quick start](docs/library-quickstart.md).

## CLI glimpse

```sh
meta config init
meta profile create default
meta auth import --input cookies.txt   # or: meta auth login
meta doctor --online
```

Details: [CLI quick start](docs/cli-quickstart.md).

## Features

- **Messenger** — text, replies, mentions, media, stickers, reactions, edits, unsend, typing/read state
- **E2EE** — persistent device state, encrypted media, receipts and mutations
- **Threads** — metadata, admins, polls, mute, group photos, DMs and contact discovery
- **Facebook** — profiles/search, posts, notifications, Marketplace and Professional Mode
- **Hardening** — encrypted secret storage, SSRF-resistant media fetch, typed models and errors

Complete matrix: [Feature matrix](docs/features.md).

## Documentation

- [Installation](docs/installation.md)
- [Library quick start](docs/library-quickstart.md)
- [CLI quick start](docs/cli-quickstart.md)
- [Authentication and profiles](docs/authentication.md)
- [E2EE state](docs/e2ee.md)
- [Security model](docs/security.md)
- [Migration](docs/migration.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Platform support](docs/support.md)
- [Maintenance policy](docs/maintenance.md)
- [Contributing](CONTRIBUTING.md)
- [Release process](RELEASING.md)

## Development

Use the Go version declared in [`go.mod`](go.mod).

```sh
git clone https://github.com/mewisme/meta.go.git
cd meta.go

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

Optional local hooks: install [pre-commit](https://pre-commit.com/), then `pre-commit install` (see [`.pre-commit-config.yaml`](.pre-commit-config.yaml)).

CLI runtime environment variables:

| Variable | Purpose |
| --- | --- |
| `META_CONFIG` | Config file path |
| `META_MASTER_KEY` | 32-byte hex/base64 master key when OS keyring is unavailable |
| `META_PROFILE` | Active profile name |

A local `go.work` that includes `../meta-extra` is supported for development and is gitignored; CI runs with `GOWORK=off`.

Release packaging uses GoReleaser (cross-platform archives, SHA-256 checksums, SBOMs, provenance). See [RELEASING.md](RELEASING.md). Third-party license metadata: [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

Contribution rules and design constraints: [CONTRIBUTING.md](CONTRIBUTING.md).

## Security, license and community

- Report vulnerabilities privately: [SECURITY.md](SECURITY.md)
- Code of Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- License: [AGPL-3.0](LICENSE)
- Changelog: [CHANGELOG.md](CHANGELOG.md)
