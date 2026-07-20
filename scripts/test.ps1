# Automated test runner for Windows PowerShell.
# Usage:
#   .\scripts\test.ps1                 # all
#   .\scripts\test.ps1 unit
#   .\scripts\test.ps1 integration
#   .\scripts\test.ps1 e2e-control

param(
    [ValidateSet("unit", "integration", "e2e-control", "all")]
    [string]$Mode = "all"
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

$env:GOPROXY = if ($env:GOPROXY) { $env:GOPROXY } else { "https://goproxy.cn,direct" }
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }

function Start-TestDeps {
    if ($env:SKIP_DOCKER -eq "1") {
        Write-Host "SKIP_DOCKER=1"
        return
    }
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "docker not found; skipping deps"
        return
    }
    Write-Host "==> starting test dependencies (Postgres :5433, Redis :6380)"
    docker compose -f examples/docker-compose/docker-compose.test.yml up -d --wait
    if (-not $env:SIPPLANE_DATABASE_URL) {
        $env:SIPPLANE_DATABASE_URL = "postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable"
    }
    if (-not $env:SIPPLANE_REDIS_ADDR) {
        $env:SIPPLANE_REDIS_ADDR = "127.0.0.1:6380"
    }
}

function Invoke-Unit {
    Write-Host "==> go test ./... (unit)"
    go test ./... -count=1 -timeout 120s
    if ($LASTEXITCODE -ne 0) { throw "unit tests failed" }
}

function Invoke-Integration {
    Start-TestDeps
    Write-Host "==> go test ./... (integration)"
    Write-Host "    DATABASE_URL=$($env:SIPPLANE_DATABASE_URL)"
    Write-Host "    REDIS_ADDR=$($env:SIPPLANE_REDIS_ADDR)"
    go test ./... -count=1 -timeout 180s
    if ($LASTEXITCODE -ne 0) { throw "integration tests failed" }
}

function Invoke-E2EControl {
    Start-TestDeps
    $port = if ($env:CONTROL_PORT) { $env:CONTROL_PORT } else { "28091" }
    $dsn = if ($env:SIPPLANE_DATABASE_URL) { $env:SIPPLANE_DATABASE_URL } else { "postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable" }
    New-Item -ItemType Directory -Force -Path bin | Out-Null
    go build -o bin/sipplane-control.exe ./cmd/sipplane-control
    go build -o bin/sipplanectl.exe ./cmd/sipplanectl
    if ($LASTEXITCODE -ne 0) { throw "build failed" }

    Write-Host "==> starting sipplane-control on :$port (with auth)"
    $env:SIPPLANE_CONTROL_TOKEN = "e2e-test-token"
    $proc = Start-Process -FilePath ".\bin\sipplane-control.exe" `
        -ArgumentList @("-listen", "127.0.0.1:$port", "-database-url", $dsn, "-seed", "examples/config", "-auth-token", "e2e-test-token") `
        -RedirectStandardOutput "bin\cp.out.log" -RedirectStandardError "bin\cp.err.log" -PassThru
    try {
        $ok = $false
        for ($i = 0; $i -lt 50; $i++) {
            try {
                $r = Invoke-WebRequest "http://127.0.0.1:$port/healthz" -UseBasicParsing -TimeoutSec 1
                if ($r.StatusCode -eq 200) { $ok = $true; break }
            } catch { Start-Sleep -Milliseconds 100 }
        }
        if (-not $ok) {
            Get-Content bin\cp.err.log -ErrorAction SilentlyContinue
            throw "control plane healthz timeout"
        }
        & .\bin\sipplanectl.exe --server "http://127.0.0.1:$port" --token "e2e-test-token" dry-run examples/config/lab.yaml | Out-Null
        & .\bin\sipplanectl.exe --server "http://127.0.0.1:$port" --token "e2e-test-token" apply examples/config/lab.yaml
        & .\bin\sipplanectl.exe --server "http://127.0.0.1:$port" --token "e2e-test-token" revision
        Write-Host "==> control-plane e2e OK"
    }
    finally {
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
}

switch ($Mode) {
    "unit" { Invoke-Unit }
    "integration" { Invoke-Integration }
    "e2e-control" { Invoke-E2EControl }
    "all" {
        Invoke-Unit
        Invoke-Integration
        Invoke-E2EControl
    }
}

Write-Host "DONE ($Mode)"
