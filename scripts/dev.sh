#!/usr/bin/env bash
# Thin wrapper kept for backwards compatibility.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec "$ROOT/scripts/test.sh" "${1:-unit}"
