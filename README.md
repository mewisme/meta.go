# meta.go

`meta.go` is a Go-first Facebook Messenger and Facebook automation library with regular Messenger and end-to-end encrypted messaging support.

Canonical module path:

```text
go.mewis.me/meta.go
```

## Highlights

- Regular Messenger: text, replies, mentions, media, stickers, forwarding, reactions, edits, unsend, typing/read state, themes, requests and Notes.
- E2EE Messenger: persistent device state, text/replies, mutations, receipts and encrypted image/video/audio/document/sticker media.
- Thread management: metadata, admins, names, emoji/nicknames, polls, mute, group photos, DM creation and Messenger contact discovery.
- Facebook: profiles/search, social actions, posts, notifications, Marketplace and Professional Mode.
- Typed models/events/errors, context-first network operations, bounded event delivery and health/error observability.
- Encrypted secret/state storage and hardened media fetching/file writes.
- Pure Go production runtime; no Python runtime or protocol subprocess is required.

See the complete [feature matrix](docs/features.md).

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

## Documentation

- [Installation](docs/installation.md)
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

```sh
go mod verify
go test ./...
go test -race ./...
go vet ./...
govulncheck ./...
```

Release packaging uses GoReleaser and produces deterministic cross-platform archives, SHA-256 checksums, SBOMs and provenance attestations. Third-party dependency license metadata is tracked in `THIRD_PARTY_LICENSES.md`.
