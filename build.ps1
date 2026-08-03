<#
  build.ps1 - build the Windows imageproxy binary into build\imageproxy.exe.
  Pure Go, no cgo. Requires Go 1.25.8+ on PATH. Run from the repo root.
#>
$ErrorActionPreference = 'Stop'
# Let explicit $LASTEXITCODE checks govern native (go) failures instead of PS 7.4+
# auto-throwing on any non-zero exit. Harmless on PS 5.1.
$PSNativeCommandUseErrorActionPreference = $false
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $here

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go toolchain not found on PATH. Install Go 1.25.8+ (https://go.dev/dl/)."
}

New-Item -ItemType Directory -Force (Join-Path $here 'build') | Out-Null

go mod download
$env:CGO_ENABLED = '0'
$env:GOOS        = 'windows'
$env:GOARCH      = 'amd64'
go build -ldflags "-s -w" -o build/imageproxy.exe ./cmd/imageproxy
if ($LASTEXITCODE -ne 0) { throw "go build failed ($LASTEXITCODE)." }

Write-Host "Built build\imageproxy.exe"

# Smoke test: start the fresh binary, poll /health-check, then stop that exact process.
$exe  = Join-Path $here 'build\imageproxy.exe'
$proc = Start-Process -FilePath $exe -ArgumentList '-addr', '127.0.0.1:8099' -PassThru -WindowStyle Hidden
$ok = $null
foreach ($i in 1..20) {
  Start-Sleep -Milliseconds 300
  try { $ok = (Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 'http://127.0.0.1:8099/health-check').Content; break } catch { }
}
if ($proc -and -not $proc.HasExited) { $proc | Stop-Process -Force }
if ($ok -eq 'OK') { Write-Host "Smoke test: health-check -> OK" }
else { Write-Warning "Smoke test did not confirm health-check (binary built OK regardless)." }
