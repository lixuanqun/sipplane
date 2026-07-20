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
mkdir -p bin
go build -o bin/sipplane ./cmd/sipplane

export SIPPLANE_HTTP_LISTEN="127.0.0.1:${HTTP_PORT}"
export SIPPLANE_ADVERTISED_PORT="${PORT}"

./bin/sipplane \
  -config examples/config/bootstrap.yaml \
  -resources examples/config \
  -listen "127.0.0.1:${PORT}" \
  -advertised-host 127.0.0.1 \
  >/tmp/sipplane-sipp-smoke.log 2>&1 &
PID=$!
cleanup() { kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; }
trap cleanup EXIT

for i in $(seq 1 80); do
  if curl -sf "http://127.0.0.1:${HTTP_PORT}/readyz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done
curl -sf "http://127.0.0.1:${HTTP_PORT}/readyz" >/dev/null

echo "==> SIPp OPTIONS"
sipp -sf examples/sipp/options_ping.xml "127.0.0.1:${PORT}" -m 1 -trace_err -nostdin
echo "==> SIPp REGISTER Digest"
sipp -sf examples/sipp/register_alice.xml "127.0.0.1:${PORT}" -m 1 -trace_err -nostdin
echo "==> SIPp smoke OK"
