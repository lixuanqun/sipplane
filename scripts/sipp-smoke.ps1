# SIPp smoke (Windows). Skips if sipp.exe is not on PATH.
param()
$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

if (-not (Get-Command sipp -ErrorAction SilentlyContinue)) {
    Write-Host "sipp not found; skip"
    exit 0
}

$env:GOPROXY = if ($env:GOPROXY) { $env:GOPROXY } else { "https://goproxy.cn,direct" }
$env:GOTOOLCHAIN = if ($env:GOTOOLCHAIN) { $env:GOTOOLCHAIN } else { "auto" }

$Port = if ($env:SIPP_SMOKE_PORT) { $env:SIPP_SMOKE_PORT } else { "15060" }
$HttpPort = if ($env:SIPP_SMOKE_HTTP_PORT) { $env:SIPP_SMOKE_HTTP_PORT } else { "18080" }
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -o bin/sipplane.exe ./cmd/sipplane

$env:SIPPLANE_HTTP_LISTEN = "127.0.0.1:$HttpPort"
$env:SIPPLANE_ADVERTISED_PORT = "$Port"

$proc = Start-Process -FilePath ".\bin\sipplane.exe" `
    -ArgumentList @("-config", "examples/config/bootstrap.yaml", "-resources", "examples/config",
        "-listen", "127.0.0.1:$Port", "-advertised-host", "127.0.0.1") `
    -RedirectStandardOutput "bin\sipp-smoke.out.log" -RedirectStandardError "bin\sipp-smoke.err.log" -PassThru
try {
    $ready = $false
    for ($i = 0; $i -lt 80; $i++) {
        try {
            $r = Invoke-WebRequest "http://127.0.0.1:$HttpPort/readyz" -UseBasicParsing -TimeoutSec 1
            if ($r.StatusCode -eq 200) { $ready = $true; break }
        } catch { Start-Sleep -Milliseconds 100 }
    }
    if (-not $ready) { throw "sipplane not ready for SIPp" }
    Write-Host "==> SIPp OPTIONS"
    & sipp -sf examples/sipp/options_ping.xml "127.0.0.1:$Port" -m 1 -trace_err -nostdin
    if ($LASTEXITCODE -ne 0) { throw "OPTIONS failed" }
    Write-Host "==> SIPp REGISTER Digest"
    & sipp -sf examples/sipp/register_alice.xml "127.0.0.1:$Port" -m 1 -trace_err -nostdin
    if ($LASTEXITCODE -ne 0) { throw "REGISTER failed" }
    Write-Host "==> SIPp smoke OK"
}
finally {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
