# E2EE state

Enable E2EE on the root client with `meta.WithE2EE(true)`.

## Persistent device state

For stable encrypted sessions across restarts, configure both a profile and a `storage.SecretStore`:

```go
client, err := meta.NewClient(
	meta.WithProfile(storage.Profile{Name: "default"}),
	meta.WithSecretStore(secretStore),
	meta.WithE2EE(true),
)
```

meta.go loads the profile's E2EE device state before connecting. If no state exists, it creates a device and persists future identity/session/prekey/sender-key mutations through the configured store.

Do not intentionally create a new device on every process start. Reusing the persisted state avoids unnecessary device churn and preserves Signal session continuity.

## Storage requirements

The application is responsible for providing an appropriate `SecretStore`. meta.go includes memory/file storage primitives and an encrypted secret-store wrapper. Persistent production state should be encrypted at rest and stored with private permissions.

Malformed or incomplete E2EE state is rejected. meta.go does not silently replace corrupted state with a new identity because that would hide state loss.

## Sending encrypted messages

Use the E2EE methods on `client.Messenger`, for example `SendE2EE`, `SendE2EEMedia`, `ReactE2EE`, `EditE2EE`, `UnsendE2EE`, `TypingE2EE` and `ReadE2EE`.

An operation that requires E2EE returns `meta.ErrE2EENotReady` when the encrypted transport is not authenticated. It does not silently downgrade to regular Messenger.

## Encrypted media

`SendE2EEMedia` supports image, video, audio/voice, document and sticker media. Incoming attachment models contain a typed `E2EEMediaReference`; pass that reference to `DownloadE2EE` to download, decrypt and verify the media hashes.

## Backup and migration

Treat E2EE state as a secret. Copy it only through the application's secret-store/migration path and never print it to logs. Migration imports do not delete the source state automatically.
