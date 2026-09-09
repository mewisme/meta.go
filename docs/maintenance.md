# Maintenance policy

## Protocol-change patches

When Facebook changes a private endpoint, persisted query, Lightspeed task or response schema:

1. reproduce with sanitized fixtures or an injected test server;
2. isolate the change in the internal protocol registry/parser/adapter;
3. add a regression test that fails on the old behavior;
4. preserve the public typed contract when possible;
5. classify incompatible remote shapes as `ErrProtocolChanged`;
6. run unit, race, vulnerability, release-config and cross-build gates;
7. ship a patch release when the public API remains compatible.

Do not scatter replacement document IDs or task labels through service packages.

## Dependency updates

For each direct dependency update:

1. review upstream changelog and license changes;
2. run `go mod tidy` and `go mod verify`;
3. update `THIRD_PARTY_LICENSES.md` when the direct dependency/version/license set changes;
4. run `govulncheck ./...`;
5. run regular and E2EE protocol regression suites;
6. rebuild the local GoReleaser snapshot and inspect SBOM/checksums;
7. re-run live smoke when the dependency touches authentication, transport, Lightspeed, GraphQL or E2EE behavior.

Protocol dependencies should be pinned to reviewed versions. Avoid unreviewed automatic major-version updates.

## Release branches and tags

Release tags are immutable. Never move or overwrite an existing `vX.Y.Z` tag. Fixes after a published release use a new patch version.
