#!/usr/bin/env bash
set -euo pipefail

if ! command -v mise >/dev/null 2>&1; then
  echo "ERROR: mise is required to build Power-Dawn." >&2
  echo "Install it from https://mise.jdx.dev/getting-started.html" >&2
  exit 1
fi

mise install
exec mise run build
