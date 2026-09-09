# Changelog

All notable changes to fbgo are documented here.

## Unreleased

### Added

- Go-first Facebook Messenger client with regular and end-to-end encrypted transports.
- Typed message, event, thread, Facebook, Marketplace, authentication and error models.
- Regular messaging with text, replies, mentions, attachments, stickers, external media, forwarding, reactions, edits, unsend, typing and read state.
- E2EE messaging with persistent device state, text, replies, reactions, edits, unsend, typing, read receipts and encrypted media.
- Thread management including metadata, admin/name/emoji/nickname changes, polls, mute, group photo, delete, DM creation and Messenger contact discovery.
- Facebook profile, search, social actions, posts, notifications, Marketplace and Professional Mode operations.
- Encrypted profile/session storage, bounded event dispatch, health/error observability and hardened media download handling.

### Security

- XChaCha20-Poly1305 secret storage with authenticated envelopes.
- SSRF-resistant media fetching with redirect validation and DNS pinning.
- Atomic private file writes and fail-closed E2EE state restoration.
