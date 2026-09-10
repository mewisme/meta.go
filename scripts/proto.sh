#!/usr/bin/env bash
set -euo pipefail

BUF_VERSION=v1.72.0
PROTOBUF_VERSION=v1.36.12
PROTOC_GEN_GO_GRPC_VERSION=v1.6.2
GOIMPORTS_VERSION=v0.50.0
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TOOLS_DIR=${XDG_CACHE_HOME:-$HOME/.cache}/meta-go/proto-tools
BIN_DIR="$TOOLS_DIR/$BUF_VERSION-$PROTOBUF_VERSION-$PROTOC_GEN_GO_GRPC_VERSION-$GOIMPORTS_VERSION"

install_tools() {
  mkdir -p "$BIN_DIR"
  [ -x "$BIN_DIR/buf" ] || GOBIN="$BIN_DIR" go install "github.com/bufbuild/buf/cmd/buf@$BUF_VERSION"
  [ -x "$BIN_DIR/protoc-gen-go" ] || GOBIN="$BIN_DIR" go install "google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOBUF_VERSION"
  [ -x "$BIN_DIR/protoc-gen-go-grpc" ] || GOBIN="$BIN_DIR" go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@$PROTOC_GEN_GO_GRPC_VERSION"
  [ -x "$BIN_DIR/goimports" ] || GOBIN="$BIN_DIR" go install "golang.org/x/tools/cmd/goimports@$GOIMPORTS_VERSION"
}

run_buf() {
  install_tools
  PATH="$BIN_DIR:$PATH" "$BIN_DIR/buf" "$@"
}

generate() {
  run_buf lint
  run_buf generate
  "$BIN_DIR/goimports" -local go.mewis.me/meta.go -w "$ROOT/gen/go/meta/v1"
}

case "${1:-check}" in
  generate)
    generate
    ;;
  lint)
    run_buf lint
    ;;
  breaking)
    baseline=${2:-.git#branch=main}
    run_buf breaking --against "$baseline"
    ;;
  check)
    generate
    git -C "$ROOT" diff --exit-code -- proto gen/go buf.yaml buf.gen.yaml
    test -z "$(git -C "$ROOT" ls-files --others --exclude-standard -- proto gen/go)"
    ;;
  *)
    echo "usage: $0 {generate|lint|breaking [baseline]|check}" >&2
    exit 2
    ;;
esac
