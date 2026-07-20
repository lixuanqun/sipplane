#!/usr/bin/env bash
# SIPp smoke: OPTIONS + Digest REGISTER against a local sipplane.
# Skips (exit 0) when `sipp` is not installed — CI installs sip-tester.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v sipp >/dev/null 2>&1; then
  echo "sipp not found; skip (install sip-tester / SIPp to enable)"
  exit 0
fi

PORT="${SIPP_SMOKE_PORT:-15060}"
HTTP_PORT="${SIPP_SMOKE_HTTP_PORT:-18080}"
LOG="${TMPDIR:-/tmp}/sipplane-sipp-smoke.$$.log"
BOOT="${TMPDIR:-/tmp}/sipplane-sipp-bootstrap.$$.yaml"
mkdir -p bin
go build -o bin/sipplane ./cmd/sipplane

cat >"$BOOT" <<EOF
listen: "127.0.0.1:${PORT}"
transport: udp
advertised_host: "127.0.0.1"
advertised_port: ${PORT}
http_listen: "127.0.0.1:${HTTP_PORT}"
config_dir: "examples/config"
realm: sipplane
log_level: error
EOF

./bin/sipplane \
  -config "$BOOT" \
  -resources examples/config \
  >"$LOG" 2>&1 &
PID=$!
cleanup() {
  kill "$PID" 2>/dev/null || true
  wait "$PID" 2>/dev/null || true
  rm -f "$BOOT"
}
trap cleanup EXIT

ready=0
for _ in $(seq 1 150); do
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "sipplane exited early; log:"
    cat "$LOG" || true
    exit 1
  fi
  if curl -sf "http://127.0.0.1:${HTTP_PORT}/readyz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 0.1
done
if [[ "$ready" != "1" ]]; then
  echo "timeout waiting for readyz on :${HTTP_PORT}; log:"
  cat "$LOG" || true
  exit 1
fi

echo "==> SIPp OPTIONS"
sipp -sf examples/sipp/options_ping.xml "127.0.0.1:${PORT}" -m 1 -trace_err -nostdin || {
  echo "OPTIONS failed; sipplane log:"; cat "$LOG" || true; exit 1
}
echo "==> SIPp REGISTER Digest"
sipp -sf examples/sipp/register_alice.xml "127.0.0.1:${PORT}" -m 1 -trace_err -nostdin || {
  echo "REGISTER failed; sipplane log:"; cat "$LOG" || true; exit 1
}
echo "==> SIPp smoke OK"
