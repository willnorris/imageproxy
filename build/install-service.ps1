<#
  install-service.ps1 - install imageproxy as a Windows service. Two methods:

    .\install-service.ps1                 # NATIVE (default): the binary self-registers
                                          #   with the SCM via `-service install` (no deps)
    .\install-service.ps1 -Method nssm    # via NSSM (nssm.exe on PATH, or pass -NssmPath)
    .\install-service.ps1 -Method nssm -NssmPath 'C:\tools\nssm\nssm.exe'

  Run from an ELEVATED PowerShell in this folder, after editing the CONFIG block.
  Compatible with Windows PowerShell 5.1 and PowerShell 7+.
  Tip: a service path WITHOUT spaces (e.g. C:\svc\imageproxy) avoids all quoting edge cases.
#>
param(
  [ValidateSet('native', 'nssm')]
  [string]$Method = 'native',
  # Full path to nssm.exe (only for -Method nssm). If omitted, nssm must be on PATH.
  [string]$NssmPath = ''
)
$ErrorActionPreference = 'Stop'
# In PowerShell 7.4+ a native command's non-zero exit throws under 'Stop'. Several calls
# below are expected to "fail" harmlessly (e.g. stopping a not-yet-installed service), so
# opt out and check $LASTEXITCODE explicitly where it matters. Harmless on PS 5.1.
$PSNativeCommandUseErrorActionPreference = $false

# -- CONFIG (edit these) -----------------------------------------------------
$ServiceName  = 'imageproxy'
$Addr         = '127.0.0.1:8080'                       # loopback: only IIS reaches it
$AllowHosts   = 'luatsumienbac.vn'                     # lock the source origin
$BaseURL      = 'https://luatsumienbac.vn/media/'      # readable URLs: /img/880x,avif/<file> resolves here
$CacheDir     = 'D:/imgcache/luatsumienbac'            # NO spaces: nssm AppParameters stores args unquoted, so a spaced path truncates -cache and drops every flag after it (incl. -baseURL). Cache is scratch; any writable no-space path works.
$Timeout      = '20s'
$SignatureKey = ''                                     # '' = unsigned (allowHosts still protects)
# ----------------------------------------------------------------------------

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$exe  = Join-Path $here 'imageproxy.exe'
if (-not (Test-Path $exe)) {
  throw "imageproxy.exe not found next to this script. Build it first: ..\build.ps1 (or see ..\DEPLOY.md)."
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
           ).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) { throw "Run this in an ELEVATED PowerShell (Administrator)." }

$logPath = Join-Path $here 'imageproxy.log'
# Runtime flags as an ARRAY. NOTE: nssm stores AppParameters UNQUOTED, so any value
# containing a space truncates that flag and Go's flag parser drops EVERY flag after
# it (this silently killed -baseURL once). Keep all values space-free; asserted below.
$svcArgs = @('-addr', $Addr, '-allowHosts', $AllowHosts, '-cache', $CacheDir, '-timeout', $Timeout, '-logFile', $logPath)
if ($BaseURL      -ne '') { $svcArgs += @('-baseURL', $BaseURL) }
if ($SignatureKey -ne '') { $svcArgs += @('-signatureKey', $SignatureKey) }

if ($Method -eq 'nssm') {
  $spaced = $svcArgs | Where-Object { $_ -match '\s' }
  if ($spaced) {
    throw ("nssm cannot store arguments containing spaces (they get truncated, dropping " +
           "every flag after them). Use space-free values (e.g. `$CacheDir = 'D:/imgcache/luatsumienbac' " +
           "and install this script under a path without spaces). Offending: " + ($spaced -join ' | '))
  }
}

# Remove any existing service first (works no matter how it was installed).
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
  Write-Host "Removing existing '$ServiceName' service ..."
  & sc.exe stop   $ServiceName 2>$null | Out-Null
  Start-Sleep -Milliseconds 600
  & sc.exe delete $ServiceName 2>$null | Out-Null
  Start-Sleep -Milliseconds 600
}

if ($Method -eq 'nssm') {
  if ($NssmPath -ne '') {
    if (-not (Test-Path $NssmPath)) { throw "NssmPath not found: $NssmPath" }
    $nssm = (Resolve-Path $NssmPath).Path
  }
  else {
    $nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
    if (-not $nssmCmd) {
      throw "nssm not found on PATH. Pass -NssmPath 'C:\tools\nssm\nssm.exe', install it (choco/scoop), or omit -Method for the native install."
    }
    $nssm = $nssmCmd.Source
  }

  Write-Host "Installing service '$ServiceName' via NSSM ..."
  # `install <name> <program> <args...>` - PS quotes each array element; nssm stores them
  # as AppParameters with correct quoting (handles the spaced/& path).
  & $nssm install $ServiceName $exe @svcArgs
  if ($LASTEXITCODE -ne 0) { throw "nssm install failed ($LASTEXITCODE)." }
  & $nssm set $ServiceName AppDirectory $here                     | Out-Null
  & $nssm set $ServiceName AppStderr (Join-Path $here 'err.log')  | Out-Null   # crash backstop; normal logs -> -logFile
  & $nssm set $ServiceName AppExit Default Restart               | Out-Null
  & $nssm set $ServiceName Start SERVICE_AUTO_START              | Out-Null
  & $nssm start $ServiceName | Out-Null
}
else {
  Write-Host "Installing service '$ServiceName' (native SCM, no nssm) ..."
  & $exe -service install @svcArgs
  if ($LASTEXITCODE -ne 0) { throw "install failed ($LASTEXITCODE)." }
  # Native crash recovery + auto-start via built-in sc.exe.
  & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000 | Out-Null
  & sc.exe config  $ServiceName start= auto | Out-Null
  & $exe -service start
}

# Confirm it came up.
Start-Sleep -Seconds 1
$svc = Get-Service $ServiceName -ErrorAction SilentlyContinue
if (-not $svc) { throw "Service '$ServiceName' was not created - see output above." }
if ($svc.Status -ne 'Running') {
  Write-Warning "Service '$ServiceName' is '$($svc.Status)', not Running. Check logs: $logPath (and Event Viewer)."
}

Write-Host ""
Write-Host "Installed '$ServiceName' via '$Method' - a real Windows service (services.msc). Status: $($svc.Status)"
Write-Host "  Query  : sc.exe query $ServiceName"
Write-Host "  Logs   : $logPath"
Write-Host "  Test   : curl.exe http://$Addr/health-check   # -> OK"
Write-Host "Uninstall (either method): .\uninstall-service.ps1"
Write-Host "Next: add the IIS /img rule + set IMAGE_RESIZER=imageproxy (see ..\DEPLOY.md)."
