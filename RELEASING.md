# Releasing meta.go

This document describes release preparation. Publishing is intentionally a separate, explicit action.

## Prerequisites

- Go version from `go.mod`.
- GoReleaser v2.18.1 or a compatible v2 release.
- Syft available on `PATH` for SBOM generation.
- A clean Git working tree.

## Local release candidate

Run the complete source gates first:

```sh
go mod verify
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...
```

Validate the release configuration:

```sh
goreleaser check
```

Build a local snapshot without publishing anything:

```sh
goreleaser release --snapshot --clean --skip=publish
```

Inspect `dist/` and verify:

- Linux, macOS and Windows archives for amd64 and arm64;
- standalone `meta-runtime_VERSION_OS_ARCH` executables for Linux/macOS and `.exe` executables for Windows, on amd64 and arm64;
- SHA-256 checksum manifest;
- SPDX JSON SBOM files;
- embedded version and commit metadata;
- archive contents include the README, license, changelog, third-party inventory and documentation.

The Node and Python SDK managed-runtime loaders default to the SDK package version when selecting a downloadable runtime. Publish an SDK version only when a runtime release with the same version exists. Installed runtime paths and explicit runtime-version overrides remain supported for independent release cadences.

## Reproducibility

The release build uses `CGO_ENABLED=0`, `-trimpath`, commit-derived timestamps and deterministic version/commit ldflags. Do not add wall-clock build timestamps to the binary.

## Publishing

Publishing is performed only from a signed/approved `vX.Y.Z` tag. Pushing such a tag starts `.github/workflows/release.yml`, which runs the source gates again, creates release artifacts, checksums and SBOMs, then creates GitHub provenance attestations.

Do not reuse or move an existing release tag. If a released build needs a fix, create a new patch version.

## Verification after publishing

Verify checksums locally:

```sh
sha256sum -c meta_VERSION_checksums.txt
```

Verify GitHub provenance with GitHub CLI:

```sh
gh attestation verify <artifact> --repo mewisme/meta.go
```

Then perform at least one regular Messenger live smoke and one E2EE live smoke using the published release candidate before declaring the version stable.
