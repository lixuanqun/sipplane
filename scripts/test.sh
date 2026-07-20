#!/usr/bin/env bash
# Cross-platform-friendly test runner for Linux/macOS/CI.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

MODE="${1:-all}" # unit | integration | all | e2e-control

start_deps() {
  if [[ "${SKIP_DOCKER:-}" == "1" ]]; then
    echo "SKIP_DOCKER=1 — assuming Postgres/Redis already available"
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker not found; running unit tests only"
    return 0
  fi
  echo "==> starting test dependencies (Postgres :5433, Redis :6380)"
  docker compose -f examples/docker-compose/docker-compose.test.yml up -d --wait
  export SIPPLANE_DATABASE_URL="${SIPPLANE_DATABASE_URL:-postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable}"
  export SIPPLANE_REDIS_ADDR="${SIPPLANE_REDIS_ADDR:-127.0.0.1:6380}"
}

run_unit() {
  echo "==> go test ./... (unit)"
  go test ./... -count=1 -timeout 120s
}

run_integration() {
  start_deps
  echo "==> go test ./... (with SIPPLANE_DATABASE_URL / SIPPLANE_REDIS_ADDR)"
  echo "    DATABASE_URL=${SIPPLANE_DATABASE_URL:-<unset>}"
  echo "    REDIS_ADDR=${SIPPLANE_REDIS_ADDR:-<unset>}"
  go test ./... -count=1 -timeout 180s
}

run_e2e_control() {
  start_deps
  local port="${CONTROL_PORT:-28091}"
  local dsn="${SIPPLANE_DATABASE_URL:-postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable}"
  echo "==> building control plane binaries"
  mkdir -p bin
  go build -o bin/sipplane-control ./cmd/sipplane-control
  go build -o bin/sipplanectl ./cmd/sipplanectl
  echo "==> starting sipplane-control on :${port} (with auth)"
  ./bin/sipplane-control -listen "127.0.0.1:${port}" -database-url "$dsn" -seed examples/config \
    -auth-token e2e-test-token \
    >bin/cp.out.log 2>bin/cp.err.log &
  local pid=$!
  trap 'kill '"$pid"' 2>/dev/null || true' EXIT
  for i in $(seq 1 50); do
    if curl -sf "http://127.0.0.1:${port}/healthz" >/dev/null; then
      break
    fi
    sleep 0.1
  done
  curl -sf "http://127.0.0.1:${port}/healthz" | grep -q ok
  ./bin/sipplanectl --server "http://127.0.0.1:${port}" --token e2e-test-token dry-run examples/config/lab.yaml >/dev/null
  ./bin/sipplanectl --server "http://127.0.0.1:${port}" --token e2e-test-token apply examples/config/lab.yaml
  ./bin/sipplanectl --server "http://127.0.0.1:${port}" --token e2e-test-token revision
  echo "==> control-plane e2e OK"
}

case "$MODE" in
  unit) run_unit ;;
  integration) run_integration ;;
  e2e-control) run_e2e_control ;;
  all)
    run_unit
    run_integration
    run_e2e_control
    ;;
  *)
    echo "usage: $0 {unit|integration|e2e-control|all}"
    exit 2
    ;;
esac

echo "DONE ($MODE)"
