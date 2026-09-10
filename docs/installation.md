# Installation

## Library

After a tagged module release is published:

```sh
go get go.mewis.me/meta.go@v1.0.0
```

The vanity import path resolves to the source repository and is the canonical import path used by `go.mod`.

## Binary archives

Release archives are built for:

- Linux amd64 and arm64;
- macOS amd64 and arm64;
- Windows amd64 and arm64.

Download the archive for your platform, verify it against the published SHA-256 checksum file, then place `meta` or `meta.exe` somewhere on `PATH`.

## Build from source

```sh
git clone https://github.com/mewisme/meta.go.git
cd meta.go
go install ./cmd/meta
```

meta.go is pure Go at runtime and does not require Python, a Python virtual environment, or an external protocol subprocess.

## Verify a build

```sh
meta version
```

Release binaries embed the release version and source commit. Development builds report `dev` and `unknown` unless ldflags are supplied.
