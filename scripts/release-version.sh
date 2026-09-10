#!/usr/bin/env bash
set -euo pipefail

tag=${1:-}
if [[ ! $tag =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "invalid release tag: $tag (expected vX.Y.Z with numeric SemVer components)" >&2
  exit 2
fi
printf '%s\n' "${tag#v}"
