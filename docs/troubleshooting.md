# Troubleshooting

## `ErrUnauthorized` or `ErrSessionExpired`

Validate that the cookie set contains `c_user` and `xs`, then run session validation. If the browser session itself is expired, obtain a new authenticated session instead of retrying indefinitely.

## `ErrCheckpointRequired`

The account requires an interactive Facebook checkpoint. Complete the checkpoint through Facebook, then refresh/import the authenticated session.

## `ErrProtocolChanged`

A private endpoint, persisted query, login flow or response shape changed. Capture only sanitized metadata such as operation name, status, error code/subcode and fbtrace ID. Do not attach cookies, tokens or full authentication bodies.

## `ErrE2EENotReady`

The encrypted transport is not authenticated. Check `client.Health().E2EE`, confirm E2EE was enabled, and verify that persisted device state is available and valid.

## Corrupt E2EE state

meta.go rejects invalid state rather than replacing it. Restore a known-good encrypted state backup. Generate a new device only when deliberately rotating the identity.

## Media download rejected

The secure downloader rejects unsafe URLs, private/special-use addresses, unsafe redirects and oversized bodies. Do not disable those checks merely to make an arbitrary URL work; download through an explicitly trusted application path if required.

## Requests time out

`meta.WithTimeout` sets the default request timeout when a caller context has no deadline. A caller-supplied context deadline takes precedence.

## Event drops

`HealthSnapshot.DroppedEventCount` increases when consumers cannot keep up with the bounded event queue. Increase the configured event buffer or make event processing faster. Avoid blocking event handlers on slow external work.

## Useful diagnostics

Inspect `client.Health()` and `meta.ClassifyError(err)`. These are designed to expose operational state without requiring secret-bearing logs.
