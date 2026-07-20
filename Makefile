# Automated testing for sipplane
#
# Usage:
#   make test              # unit tests (no external deps required)
#   make test-integration  # unit + Postgres/Redis when available
#   make test-e2e          # control-plane process e2e
#   make test-sipp         # SIPp OPTIONS + REGISTER (skips if no sipp)
#   make test-compose      # docker compose.test validate + up
#   ./scripts/test.sh      # Linux/macOS
#   ./scripts/test.ps1     # Windows PowerShell
#
# Optional env:
#   SIPPLANE_DATABASE_URL  postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable
#   SIPPLANE_REDIS_ADDR    127.0.0.1:6380
#   SKIP_DOCKER=1          do not start compose.test deps

.PHONY: test test-unit test-integration test-e2e test-sipp test-compose test-race deps-up deps-down build

export GOPROXY ?= https://goproxy.cn,direct
export GOTOOLCHAIN ?= auto

ifeq ($(OS),Windows_NT)
EXE := .exe
endif

test: test-unit

test-unit:
	go test ./... -count=1

test-race:
	go test ./... -count=1 -race

test-integration:
	@$(MAKE) deps-up
	@SIPPLANE_DATABASE_URL=$${SIPPLANE_DATABASE_URL:-postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable} \
	 SIPPLANE_REDIS_ADDR=$${SIPPLANE_REDIS_ADDR:-127.0.0.1:6380} \
	 go test ./... -count=1 -timeout 120s
	@echo "integration tests finished"

test-e2e:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -File scripts/test.ps1 e2e-control
else
	./scripts/test.sh e2e-control
endif

test-sipp:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -File scripts/sipp-smoke.ps1
else
	bash scripts/sipp-smoke.sh
endif

test-compose:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -File scripts/compose-smoke.ps1
else
	bash scripts/compose-smoke.sh
endif

deps-up:
	@if [ "$${SKIP_DOCKER}" = "1" ]; then echo "SKIP_DOCKER=1"; exit 0; fi
	@docker compose -f examples/docker-compose/docker-compose.test.yml up -d --wait

deps-down:
	docker compose -f examples/docker-compose/docker-compose.test.yml down -v || true

build:
	go build -o bin/sipplane$(EXE) ./cmd/sipplane
	go build -o bin/sipplane-control$(EXE) ./cmd/sipplane-control
	go build -o bin/sipplanectl$(EXE) ./cmd/sipplanectl

run:
	go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config

control:
	go run ./cmd/sipplane-control -listen 127.0.0.1:8090 -seed examples/config
