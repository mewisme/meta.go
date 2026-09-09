# Platform support

Release builds target the following combinations with `CGO_ENABLED=0`:

| OS | Architecture | Build verification | Runtime smoke status |
| --- | --- | --- | --- |
| Linux | amd64 | Yes | Tested |
| Linux | arm64 | Yes | Build-only |
| macOS | amd64 | Yes | Build-only |
| macOS | arm64 | Yes | Build-only |
| Windows | amd64 | Yes | Build-only |
| Windows | arm64 | Yes | Build-only |

`Build-only` means cross-compilation and package tests pass for the target, but the current release candidate has not been exercised end-to-end on that host/architecture.

The core runtime is pure Go and does not require Python. OS keyring behavior depends on the host desktop/session facilities; applications can provide their own `storage.SecretStore` when the native keyring is unavailable.

Before changing a target from build-only to tested, run at least:

1. clean install/startup;
2. session validation and regular connection;
3. event receive smoke;
4. regular send + cleanup;
5. E2EE connect/state reuse when E2EE is supported by that deployment.
