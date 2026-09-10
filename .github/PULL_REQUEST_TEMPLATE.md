## Summary

<!-- What does this PR change and why? -->

## Test plan

- [ ] `go test ./...`
- [ ] `go test -race ./...` (when touching concurrency, networking or events)
- [ ] `gofmt` / tidy clean (`test -z "$(gofmt -l .)"`, `go mod tidy`)

## Checklist

- [ ] No cookies, passwords, TOTP seeds, tokens or E2EE key material in the diff or issue links
- [ ] Tests cover new or changed behavior (including negative cases for auth/storage/media/E2EE)
- [ ] Direct dependency changes update `THIRD_PARTY_LICENSES.md`
- [ ] Release/build metadata changes update `.goreleaser.yaml` / `RELEASING.md` when applicable
