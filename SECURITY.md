# Security Policy

## Supported versions

Security fixes are applied to the latest published release line of meta.go. Older tags are not backported unless a maintainer explicitly announces otherwise. See [docs/maintenance.md](docs/maintenance.md) for release and dependency policy.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Report privately via [GitHub Security Advisories](https://github.com/mewisme/meta.go/security/advisories/new) for the `mewisme/meta.go` repository.

Repository maintainers should keep [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) enabled in the GitHub repository settings so that form is available.

### What to include

- Affected version or commit
- Impact summary (authentication bypass, secret exposure, SSRF, E2EE state integrity, etc.)
- Steps to reproduce with **sanitized** fixtures or a minimal PoC
- Any known workarounds

### What never to include

Do not attach or paste:

- Session cookies (`c_user`, `xs`, and related values)
- Account passwords or TOTP seeds / one-time codes
- Access tokens
- Private E2EE device state, identity keys or ADV secrets
- Live account credentials of any kind

Use redacted logs and fake IDs in reproductions.

## Response

Maintainers aim to acknowledge private reports within a few business days and to triage severity and scope after that. Fix timelines depend on impact and whether a coordinated disclosure is needed.

Public issues and pull requests must not include secrets. See also the product [security model](docs/security.md).
