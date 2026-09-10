# Migration

meta.go is designed to replace an existing Python Messenger integration at the feature level without requiring the Python runtime architecture.

## Principles

- Migrate behavior and persisted state deliberately; do not mirror process/subprocess structure.
- Import existing authentication/session material without deleting the source files.
- Validate the imported account before enabling mutations.
- Preserve E2EE device state when continuity matters; do not create a fresh encrypted device merely because the runtime language changed.

## Cookies

Existing cookie strings can be imported safely:

```go
manager := auth.ProfileManager{Profiles: profiles, Secrets: secrets}
if err := manager.ImportLegacyCookieString(ctx, "default", rawCookieString); err != nil {
	return err
}
```

`ImportLegacyCookieString` refuses to overwrite an existing cookie secret.

For JSON exports with supported cookie fields, use `ImportLegacyJSON`. Keep the original export until the new profile has been validated successfully.

## E2EE device state

Use `ImportLegacyE2EEState` to stage an existing device-state blob into a profile. The import refuses to overwrite an existing E2EE state entry.

The runtime loader accepts the supported compatibility state representation and converts it into meta.go's versioned device store. Corrupt state fails closed.

## Application code

Replace dynamic dictionaries/callback payloads with typed meta.go models. Typical mappings are:

- one client instance per account/profile;
- `client.Messenger` for regular and encrypted messaging;
- `client.Threads` for thread operations;
- `client.Facebook` for Facebook operations;
- `client.Events()` or `client.On` for realtime events;
- `errors.Is` plus `meta.ClassifyError` for error handling.

All network operations should inherit the application's `context.Context` and cancellation policy.

## Cutover sequence

1. Back up the existing cookies and E2EE state.
2. Create a new meta.go profile and import state without deleting the source.
3. Validate the session and connect read-only.
4. Confirm account identity, thread reads and realtime events.
5. Confirm a regular send/cleanup smoke.
6. If E2EE is used, confirm the persisted device reconnects and an encrypted smoke succeeds.
7. Switch production traffic only after both runtime and persistence checks succeed.

Rollback is simply returning to the previous runtime with its original untouched state.
