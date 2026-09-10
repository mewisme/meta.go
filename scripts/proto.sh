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

generate_node() {
  image_dir=$(mktemp -d)
  image="$image_dir/image.binpb"
  generated="$ROOT/sdk/node/src/gen"
  backup_dir=$(mktemp -d)
  had_generated=0
  status=0
  if [ -d "$generated" ]; then
    cp -R "$generated" "$backup_dir/gen"
    had_generated=1
  fi
  if (
    cd "$ROOT" &&
    run_buf lint &&
    run_buf build --exclude-source-info -o "$image" &&
    run_buf generate "$image" --template sdk/node/buf.gen.yaml
  ); then
    status=0
  else
    status=$?
    rm -rf "$generated"
    if [ "$had_generated" -eq 1 ]; then
      cp -R "$backup_dir/gen" "$generated"
    fi
  fi
  rm -rf "$image_dir" "$backup_dir"
  return "$status"
}

generate_python() {
  image_dir=$(mktemp -d)
  image="$image_dir/image.binpb"
  generated="$ROOT/sdk/python/.generated"
  status=0
  rm -rf "$generated"
  if (
    cd "$ROOT" &&
    run_buf lint &&
    run_buf build --exclude-source-info -o "$image" &&
    run_buf generate "$image" --template sdk/python/buf.gen.yaml &&
    python3 - <<'PY_NORMALIZE'
from pathlib import Path
for path in Path("sdk/python/.generated/meta").rglob("*"):
    if path.is_file() and path.suffix in {".py", ".pyi"}:
        path.write_text(path.read_text().rstrip() + "\n")
PY_NORMALIZE
  ); then
    rm -rf sdk/python/src/meta
    mkdir -p sdk/python/src
    cp -R sdk/python/.generated/meta sdk/python/src/meta
    touch sdk/python/src/meta/__init__.py sdk/python/src/meta/py.typed sdk/python/src/meta/v1/__init__.py
  else
    status=$?
  fi
  rm -rf "$image_dir" "$generated"
  return "$status"
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
  generate-node)
    generate_node
    ;;
  check-node)
    generate_node
    git -C "$ROOT" diff --exit-code -- sdk/node/src/gen
    test -z "$(git -C "$ROOT" ls-files --others --exclude-standard -- sdk/node/src/gen)"
    ;;
  generate-python)
    generate_python
    ;;
  check-python)
    generate_python
    git -C "$ROOT" diff --exit-code -- sdk/python/src/meta
    test -z "$(git -C "$ROOT" ls-files --others --exclude-standard -- sdk/python/src/meta)"
    ;;
  check)
    generate
    git -C "$ROOT" diff --exit-code -- proto gen/go buf.yaml buf.gen.yaml
    test -z "$(git -C "$ROOT" ls-files --others --exclude-standard -- proto gen/go)"
    ;;
  *)
    echo "usage: $0 {generate|generate-node|generate-python|lint|breaking [baseline]|check|check-node|check-python}" >&2
    exit 2
    ;;
esac
