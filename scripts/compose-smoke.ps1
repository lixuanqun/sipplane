# Compose smoke (Windows): validate + optional up of test deps.
param(
    [switch]$ConfigOnly
)
$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root
$Compose = "examples/docker-compose/docker-compose.test.yml"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "docker not found; skip"
    exit 0
}

Write-Host "==> docker compose config"
docker compose -f $Compose config -q
if ($LASTEXITCODE -ne 0) { throw "compose config failed" }

if ($ConfigOnly -or $env:COMPOSE_SMOKE_UP -eq "0") {
    Write-Host "==> compose config OK (up skipped)"
    exit 0
}

Write-Host "==> docker compose up"
docker compose -f $Compose up -d --wait
docker compose -f $Compose exec -T postgres pg_isready -U sipplane -d sipplane
$pong = docker compose -f $Compose exec -T redis redis-cli ping
if ($pong -notmatch "PONG") { throw "redis ping failed: $pong" }
Write-Host "==> compose smoke OK"
