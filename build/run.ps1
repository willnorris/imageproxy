<#
  run.ps1 - run imageproxy in the FOREGROUND for testing (Ctrl+C to stop).
  Same config as the service; edit to taste. For production use install-service.ps1.
#>
$ErrorActionPreference = 'Stop'
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$exe  = Join-Path $here 'imageproxy.exe'
if (-not (Test-Path $exe)) { throw "imageproxy.exe not found. Build it first (..\build.ps1 or ..\DEPLOY.md)." }

& $exe -addr 127.0.0.1:8080 `
       -allowHosts luatsumienbac.vn `
       -baseURL https://luatsumienbac.vn/media/ `
       -cache D:/imgcache/luatsumienbac `
       -timeout 20s `
       -verbose
