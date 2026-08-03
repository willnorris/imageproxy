# imageproxy — Windows deploy bundle

Copy this folder to the server (e.g. `C:\svc\imageproxy\`) and follow the checklist.
`imageproxy.exe` itself is **not** committed to git (built artifact) — produce it with
`..\build.ps1`. Full reference: [../DEPLOY.md](../DEPLOY.md).

## Contents
| File | Purpose |
|------|---------|
| `imageproxy.exe` | the resizer binary (built by `..\build.ps1`; not in git) |
| `install-service.ps1` | install as a Windows service — **native** (default) or **`-Method nssm`**; edit CONFIG first |
| `uninstall-service.ps1` | stop + remove the service (works for either method) |
| `run.ps1` | run in the foreground for testing |
| `config.example.env` | env-var alternative to CLI flags |

> **Two install methods, one script.** Default is **native** — the binary self-registers
> with the Windows SCM (no third-party tool). If you have `nssm.exe`, run
> `install-service.ps1 -Method nssm` instead. Both produce a real service in
> `services.msc`; `uninstall-service.ps1` removes either.

## Deploy checklist (Windows)
1. **Build** the exe (on a machine with Go): from the repo root run `.\build.ps1`
   → produces `build\imageproxy.exe`.
2. **Copy** this `build\` folder to the server, e.g. `C:\svc\imageproxy\`.
3. **Edit** `install-service.ps1` → the CONFIG block (cache dir, allowHosts, port, optional key).
4. **Install** (elevated PowerShell) — pick one:
   - native (default): `powershell -ExecutionPolicy Bypass -File .\install-service.ps1`
   - nssm: `powershell -ExecutionPolicy Bypass -File .\install-service.ps1 -Method nssm`
   → registers + starts the service with auto-start + crash-restart.
5. **Smoke test:** `curl.exe http://127.0.0.1:8080/health-check` → `OK`  (logs: `imageproxy.log`)
6. **IIS:** add the reverse-proxy rule (needs URL Rewrite + ARR, proxy enabled):
   ```xml
   <rule name="ImageProxy" stopProcessing="true">
     <match url="^img/(.*)" />
     <action type="Rewrite" url="http://localhost:8080/{R:1}" />
   </rule>
   ```
7. **Build env:** set `IMAGE_RESIZER=imageproxy` so the site emits `/img/...` URLs.
8. **Verify live:** `curl.exe -I "https://luatsumienbac.vn/img/800x,avif,q55/<b64url-of-a-/media-file>"`
   → `200`, `Content-Type: image/avif`.

Linux deployment (systemd + nginx/Caddy) is in [../DEPLOY.md](../DEPLOY.md).
