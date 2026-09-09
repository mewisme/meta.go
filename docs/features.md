# Feature matrix

This matrix describes the supported library capabilities of fbgo.

## Authentication and runtime

- Cookie parsing/import and browser-cookie JSON import.
- Session validation and compatibility bootstrap.
- Identifier/password login with TOTP seed or pre-generated OTP.
- Typed auth/checkpoint/rate-limit/protocol errors.
- Profile/session/E2EE-state persistence and transactional profile rename.
- Context-first connection lifecycle, bounded events, health snapshots and error classification.

## Regular Messenger

- Text, replies and mentions.
- Image, GIF, video, audio and file upload/send.
- Stickers and external media URLs.
- Forward message.
- Reaction add/remove, edit and unsend.
- Typing and read state.
- Message requests.
- Theme list/find/set.
- Notes read/create/delete/recreate.
- Hardened media download.

## End-to-end encrypted Messenger

- Direct registration/connect and persistent device state.
- Text/reply send.
- Reaction, edit, unsend, typing and read receipt.
- Image, video, audio/voice, document and sticker media.
- Encrypted media download/decrypt/hash verification.
- Typed encrypted message/reaction/edit/unsend/receipt events.
- Fail-closed behavior when the encrypted transport or state is not ready.

## Threads and Messenger contacts

- List/get thread metadata and sync sequence.
- Admin, name, emoji and nickname mutations.
- Poll creation and voting.
- Mute/unmute/indefinite mute.
- Group photo update.
- Thread deletion and direct-message creation.
- Messenger user search and contact detail.

## Facebook

- User/profile detail and search.
- Bio update and additional profile creation.
- Unfriend and block/unblock.
- Post create/archive/delete.
- Notifications.
- Marketplace listing create/detail and typed category registry.
- Professional Mode enable/disable.

## Architecture differences

fbgo implements these capabilities directly in Go. Process bridges, Python subprocesses and runtime JSON-RPC layers are not part of the production architecture.
