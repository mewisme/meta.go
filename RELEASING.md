# Releasing meta.go

One repository tag publishes the Go CLI/runtime and both supported SDKs. The tag is the only release version source.

## Version policy

Release tags must be numeric SemVer in the form `vX.Y.Z`. The current date-based convention fits this directly, for example:

```text
v0.260911.0 -> Go/GitHub v0.260911.0 -> npm/PyPI 0.260911.0
```

Validate a tag locally with:

```sh
bash scripts/release-version.sh v0.260911.0
```

SDK source manifests intentionally stay at `0.0.0`. Release jobs materialize the tag-derived version into temporary CI checkouts; release version bumps are not committed.

## Published artifacts

One tag publishes:

- GitHub/Go CLI archives for Linux, macOS and Windows on amd64/arm64;
- standalone `meta-runtime_VERSION_OS_ARCH` GitHub binaries from the same runtime build matrix;
- `@meewmeew/meta`;
- six `@meewmeew/meta-runtime-*` npm packages;
- `mewisme-meta`;
- eight `mewisme-meta-runtime` wheels: macOS/Windows x64+arm64 plus manylinux/musllinux x64+arm64.

The runtime binary is built once per OS/architecture. The same binary is reused for the npm platform package, Python runtime wheel and standalone GitHub asset.

## Package-manager runtime behavior

`@meewmeew/meta` pins all six optional runtime packages to its exact version. npm installs only the package matching the host `os`/`cpu` constraints.

`mewisme-meta` pins `mewisme-meta-runtime` to its exact version. pip selects the compatible platform wheel. Linux publishes both manylinux and musllinux tags from the same statically linked Go binary.

Neither SDK performs a first-launch runtime download. Explicit runtime paths, `META_RUNTIME_PATH`, and an installed `meta-runtime` on `PATH` remain supported.

## Registry trusted publishers

The release workflow uses OIDC Trusted Publishing and does not require long-lived npm/PyPI write tokens. Configure these registry-side publishers before the first release:

- npm environment `npm`: `@meewmeew/meta` and all six `@meewmeew/meta-runtime-*` packages;
- PyPI environment `pypi`: `mewisme-meta` and `mewisme-meta-runtime`;
- repository: `mewisme/meta.go`;
- workflow filename: `release.yml`.

Protect the `npm` and `pypi` GitHub environments with required reviewers if desired, and protect `v*` tags from unauthorized creation/movement.

## Local source gate

Before tagging:

```sh
go mod verify
test -z "$(gofmt -l .)"
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
govulncheck ./...
pnpm -C sdk/node check
uv run --directory sdk/python ruff check src/mewisme_meta tests scripts runtime/src
uv run --directory sdk/python mypy src/mewisme_meta
uv run --directory sdk/python pytest
goreleaser check
```

The CI release workflow repeats these gates before any publish job receives OIDC permission.

## Release workflow

Pushing `vX.Y.Z` starts `.github/workflows/release.yml`:

```text
validate tag/source
  -> build runtime once for 6 OS/arch targets
     -> package 6 npm runtime packages -> validate -> publish
     -> package 8 Python runtime wheels -> validate -> publish
  -> package Node SDK -------------------------------> publish after npm runtimes
  -> package Python SDK -----------------------------> publish after PyPI runtimes
  -> GoReleaser CLI + standalone runtime upload after both SDKs publish
```

Runtime packages are published before their root SDK, so a newly published root package never references a runtime version that has not been published yet.

## Local release-artifact validation

Release preparation helpers accept an explicit version:

```sh
node sdk/node/scripts/prepare-release.mjs 0.260911.0
python3 sdk/python/scripts/prepare-release.py 0.260911.0
python3 scripts/validate-release.py --version 0.260911.0 --node-dir sdk/node --python-dir sdk/python
```

Run those only in a disposable checkout because they modify package manifests in place. CI always uses disposable tag checkouts.

`scripts/validate-release.py` also validates packed Node/Python SDK artifacts and complete runtime package/wheel sets before publishing.

## Go/GitHub release

GoReleaser builds only the `meta` CLI archives. Runtime binaries are intentionally excluded from GoReleaser so they cannot diverge from the copies installed by SDK packages. The workflow uploads the prebuilt runtime matrix to the same GitHub release and creates a separate SHA-256 runtime checksum manifest.

Release artifacts receive GitHub attestations; npm publishing uses provenance and OIDC; PyPI uses Trusted Publishing.

Do not reuse or move an existing release tag. If a released build needs a fix, create a new patch version.

## Verification after publishing

Verify GitHub checksums and provenance, then install the released SDKs in clean environments and ensure managed mode starts the bundled runtime without network fallback. Finally perform at least one regular Messenger live smoke and one E2EE live smoke before declaring the version stable.
