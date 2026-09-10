# Security model

meta.go interacts with authenticated Facebook and Messenger sessions. Cookies, passwords, TOTP seeds, one-time codes and E2EE device state must be treated as secrets.

## Secrets at rest

Persistent applications should keep secrets behind `storage.SecretStore`. meta.go's encrypted secret-store layer uses XChaCha20-Poly1305 with authenticated associated data so ciphertext cannot be moved between profile/key slots without detection.

Secret/profile files use private permissions and atomic replacement. Applications should additionally protect the operating-system account and any keyring/master-key material.

## Logging

The default structured logger is wrapped with redaction for common credential/session keys. Do not add raw cookies, passwords, access tokens, TOTP seeds, E2EE keys or full authentication responses to application logs.

`meta.ClassifyError` provides low-cardinality error categories suitable for metrics without using raw server error text as a label.

## Media downloads

The hardened media downloader:

- accepts HTTP(S) only;
- rejects embedded URL credentials;
- rejects loopback, private, link-local, multicast, CGNAT, benchmark and documentation-only address ranges;
- disables environment HTTP proxies for protected media fetches;
- resolves and pins public IPs at dial time;
- validates every redirect;
- enforces size limits;
- writes file output through temporary-file + fsync + atomic rename.

Applications that bypass these helpers are responsible for equivalent SSRF and file-safety controls.

## E2EE

E2EE device state is fail-closed. Invalid registration IDs, zero private keys, malformed ADV secrets, missing identities and corrupt snapshots are rejected instead of silently generating replacement state.

E2EE-required operations do not downgrade to regular Messenger.

## Dependency and vulnerability policy

Release gates run `govulncheck` against reachable symbols and maintain a dependency/license inventory. A module-only advisory is not treated as reachable application code, but it must be reviewed and documented before release.

## Reporting

Report security vulnerabilities privately using the process in [`SECURITY.md`](../SECURITY.md). Do not open public issues for vulnerabilities.

Do not include session cookies, account passwords, TOTP seeds, access tokens or private E2EE state in bug reports or public issue attachments.
