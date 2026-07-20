#!/usr/bin/env bash
# Compose smoke: validate and (optionally) start test deps.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
COMPOSE="examples/docker-compose/docker-compose.test.yml"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found; skip"
  exit 0
fi

echo "==> docker compose config"
docker compose -f "$COMPOSE" config -q

if [[ "${COMPOSE_SMOKE_UP:-1}" == "0" ]]; then
  echo "==> compose config OK (up skipped)"
  exit 0
fi

echo "==> docker compose up"
docker compose -f "$COMPOSE" up -d --wait

echo "==> probe Postgres :5433"
docker compose -f "$COMPOSE" exec -T postgres pg_isready -U sipplane -d sipplane

echo "==> probe Redis :6380"
docker compose -f "$COMPOSE" exec -T redis redis-cli ping | grep -qi pong

echo "==> compose smoke OK"
# leave stack running for subsequent integration tests; caller may deps-down
