# Deploying imageproxy (trungnsbkvn fork) — Windows & Linux

Careful, end-to-end deployment guide for the WebP/AVIF-enabled fork that powers the
`/img` on-the-fly resizer for **luatsumienbac.vn**. The fork is a **single pure-Go
binary** (no cgo, no libvips, no Docker) that builds and runs natively on **Windows**
and **Linux**. For what the fork adds over upstream, see [FORK_NOTES.md](FORK_NOTES.md).

> Site-side wiring (the Astro `IMAGE_RESIZER` env, the `<picture>` output, the media
> migration) lives in the site repo: `docs/MEDIA_RESIZER.md` and
> `docs/RESIZER_PROVIDERS.md`. This file is only about running the engine.

---

## 1. How it fits together

```
Browser ──/img/{opts}/{b64url(mediaURL)}──▶ reverse proxy (IIS / nginx / Caddy)
                                             strips /img, forwards to 127.0.0.1:8080
                                                   │
                                                   ▼
                                            imageproxy (this binary)
                                            • fetches the ORIGINAL over HTTP from
                                              https://luatsumienbac.vn/media/<file>
                                            • resizes + encodes AVIF/WebP/JPEG
                                            • caches the RESULT to disk (encode once)
                                                   │
                                                   ▼
                                             Cloudflare caches at the edge (immutable)
```

- imageproxy **fetches originals over HTTP** from the public `/media` URL, so it needs
  **no access to `D:\media`** — it only needs outbound HTTP to your own origin.
- It listens on **loopback only** (`127.0.0.1:8080`); the public reverse proxy is the
  only thing that can reach it.
- The `-cache` disk store holds **transformed** variants (each width×format encoded
  once), so repeat hits are served straight from disk; Cloudflare caches downstream.

---

## 2. Build the binary

Requires **Go 1.25.8+** (built & tested with Go 1.26). No C toolchain needed.

```bash
git clone https://github.com/trungnsbkvn/imageproxy.git
cd imageproxy
go mod tidy          # first time only; pulls gen2brain webp/avif + wazero
```

**Linux (native):**
```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o imageproxy ./cmd/imageproxy
```

**Windows (native, PowerShell):**
```powershell
$env:CGO_ENABLED=0
go build -ldflags "-s -w" -o imageproxy.exe .\cmd\imageproxy
```

**Cross-compile** (build once, ship anywhere — pure Go makes this trivial):
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o imageproxy.exe ./cmd/imageproxy
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w" -o imageproxy      ./cmd/imageproxy
```

Output is a single ~48 MB static binary (it embeds the libwebp + libaom WASM blobs).
`-ldflags "-s -w"` strips debug info to shrink it. Verify:
```bash
./imageproxy -addr 127.0.0.1:8080 &      # or .\imageproxy.exe on Windows
curl http://127.0.0.1:8080/health-check  # -> OK
```

---

## 3. Configuration reference

Every flag can also be set as an environment variable prefixed **`IMAGEPROXY_`** with
the flag name upper-cased (via `envy`). Env vars are convenient for services.

| Flag | `IMAGEPROXY_` env | Recommended value | Purpose |
|------|-------------------|-------------------|---------|
| `-addr` | `IMAGEPROXY_ADDR` | `127.0.0.1:8080` | listen address (loopback → only the proxy reaches it). `unix:/path` also supported. |
| `-allowHosts` | `IMAGEPROXY_ALLOWHOSTS` | `luatsumienbac.vn` | **lock the source origin** — prevents open-proxy abuse. Comma-separated. |
| `-cache` | `IMAGEPROXY_CACHE` | a disk path (see §6) | cache transformed variants. Omit → no cache (re-encodes every hit). |
| `-signatureKey` | `IMAGEPROXY_SIGNATUREKEY` | *(optional)* | HMAC key for signed URLs (see §7). `@/path` reads the key from a file. |
| `-contentTypes` | `IMAGEPROXY_CONTENTTYPES` | `image/*` (default) | allowed source content types. Leave default. |
| `-timeout` | `IMAGEPROXY_TIMEOUT` | `20s` | per-request limit. `0` = none. |
| `-verbose` | `IMAGEPROXY_VERBOSE` | `false` in prod | debug logging (useful during first bring-up). |
| `-scaleUp` | `IMAGEPROXY_SCALEUP` | `false` | never upscale past the original. Keep false. |
| `-service` | — | *(control)* | fork addition: `install`/`uninstall`/`start`/`stop`/`restart` the OS service, then exit. |
| `-logFile` | `IMAGEPROXY_LOGFILE` | a file path | fork addition: append logs to a file (a service's stdout is discarded — set this). |

Endpoints: `GET /health-check` → `OK`; `GET /metrics` → Prometheus.

> **Server write timeout:** the HTTP server uses a 30 s write timeout. AVIF encoding of
> very large originals can be slow — but the CMS already downsizes uploads to ~1600 px,
> so encodes stay a few seconds, well under the limit. Keep originals reasonably sized.

---

## 4. Run as a service

### 4a. Windows — as a service (native by default, or nssm)
Two methods, both giving a real service in `services.msc`. **Native** is the default and
needs no third-party tool; **nssm** is available if you prefer it. The bundled
`build\install-service.ps1` does either from a CONFIG block (`-Method nssm` for the
latter) — prefer it over the raw commands below.

**Native** — the binary registers **itself** with the Service Control Manager. Put it
somewhere stable, e.g. `C:\svc\imageproxy\`, and from an **elevated** PowerShell:

```powershell
$exe = "C:\svc\imageproxy\imageproxy.exe"

# Install: everything after `-service install` becomes the service's command line.
& $exe -service install `
    -addr 127.0.0.1:8080 `
    -allowHosts luatsumienbac.vn `
    -cache D:/media/luatsumienbac/_imgcache `
    -timeout 20s `
    -logFile C:\svc\imageproxy\imageproxy.log

# Native crash recovery + auto-start, via built-in sc.exe:
sc.exe failure imageproxy reset= 86400 actions= restart/5000/restart/5000/restart/5000
sc.exe config  imageproxy start= auto

& $exe -service start
sc.exe query imageproxy          # or services.msc
```
Manage it with `imageproxy.exe -service stop|start|restart|uninstall` (or the usual
`sc.exe` / `services.msc`).

**NSSM** (alternative).

*Get nssm onto the server* — it's a single standalone binary, nothing to install, no
runtime:
- **Chocolatey:** `choco install nssm`  ·  **Scoop:** `scoop install nssm` (both add it to PATH)
- **Manual:** download `nssm-2.24.zip` from <https://nssm.cc/download> (or a mirror), unzip,
  take **`win64\nssm.exe`**. Then either add its folder to PATH, or skip PATH entirely and
  point the installer at it:
  ```powershell
  .\install-service.ps1 -Method nssm -NssmPath 'C:\tools\nssm\nssm.exe'
  ```
  Verify with `nssm version`.

*Configure by hand* (what `install-service.ps1 -Method nssm` does for you):
```powershell
$exe = "C:\svc\imageproxy\imageproxy.exe"
# Pass the flags right after the program — nssm stores them as AppParameters, quoting
# spaced paths itself (more robust than `set AppParameters "...escaped quotes..."`):
nssm install imageproxy $exe -addr 127.0.0.1:8080 -allowHosts luatsumienbac.vn -cache "D:/media/luatsumienbac/_imgcache" -timeout 20s -logFile "C:\svc\imageproxy\imageproxy.log"
nssm set imageproxy AppDirectory "C:\svc\imageproxy"
nssm set imageproxy AppExit Default Restart          # restart if the process exits
nssm set imageproxy Start SERVICE_AUTO_START         # start at boot
nssm set imageproxy AppStderr "C:\svc\imageproxy\err.log"   # crash backstop (normal logs -> -logFile)
nssm start imageproxy
```
`nssm edit imageproxy` opens a GUI for these; `nssm restart imageproxy` after changes. On
stop, nssm sends the process Ctrl+C first, so imageproxy shuts down gracefully. Runs as
LocalSystem unless you set `nssm set imageproxy ObjectName <user> <password>`.

Either method: remove with `build\uninstall-service.ps1` (or `sc.exe delete imageproxy`,
which works for both). The two methods are interchangeable — don't run both at once.

> **Logs:** a service's stdout is discarded by Windows, so pass `-logFile` (both methods)
> to capture startup + errors. Start/stop/failure are also in the Windows Event Log.

> **Cache path on Windows:** both `D:/media/.../_imgcache` and `D:\media\...\_imgcache`
> are accepted (verified). Forward slashes are slightly safer (the value passes through
> Go's `url.Parse`). The cache directory is created on first use.

> **Firewall:** binding to `127.0.0.1` already blocks external access. No inbound rule
> is needed — only IIS on the same box connects to it. The service runs as LocalSystem
> by default (can read `D:\media` and write the cache).

### 4b. Linux — via systemd
> The binary can also self-install on Linux: `sudo imageproxy -service install -addr … -cache …`
> writes a systemd unit automatically (same `-service` flag as Windows). The hand-written
> unit below is preferred in production because it adds a dedicated user + hardening.

Create a dedicated unprivileged user and a unit.

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin imageproxy
sudo install -Dm755 imageproxy /usr/local/bin/imageproxy
sudo install -d -o imageproxy -g imageproxy /var/cache/imageproxy
```

`/etc/imageproxy.env`:
```ini
IMAGEPROXY_ADDR=127.0.0.1:8080
IMAGEPROXY_ALLOWHOSTS=luatsumienbac.vn
IMAGEPROXY_CACHE=/var/cache/imageproxy
IMAGEPROXY_TIMEOUT=20s
# IMAGEPROXY_SIGNATUREKEY=...        # optional, see §7
```

`/etc/systemd/system/imageproxy.service`:
```ini
[Unit]
Description=imageproxy (WebP/AVIF resizer)
After=network-online.target
Wants=network-online.target

[Service]
User=imageproxy
Group=imageproxy
EnvironmentFile=/etc/imageproxy.env
ExecStart=/usr/local/bin/imageproxy
Restart=on-failure
RestartSec=2
# hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/cache/imageproxy

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now imageproxy
systemctl status imageproxy
curl http://127.0.0.1:8080/health-check   # -> OK
```

---

## 5. Reverse proxy (route `/img` → imageproxy)

The site emits `/img/{opts}/{b64url}`. imageproxy has **no `/img` prefix**, so the proxy
must **strip `/img`** before forwarding.

### 5a. Windows IIS (URL Rewrite + ARR)
Requires the **URL Rewrite** and **Application Request Routing (ARR)** modules, and ARR
proxy enabled (IIS Manager → server node → *Application Request Routing Cache* → *Server
Proxy Settings* → **Enable proxy**). Then in the site's `web.config`, inside
`<system.webServer><rewrite><rules>`:

```xml
<rule name="ImageProxy" stopProcessing="true">
  <match url="^img/(.*)" />
  <action type="Rewrite" url="http://localhost:8080/{R:1}" />
</rule>
```
`{R:1}` is everything after `img/`, so `http://localhost:8080/{opts}/{b64url}` — the
`/img` prefix is dropped, which is exactly what imageproxy expects. Put this rule
**before** any catch-all rule and after the `/media` static rule.

### 5b. Linux nginx
```nginx
location /img/ {
    proxy_pass http://127.0.0.1:8080/;   # trailing slash strips the /img/ prefix
    proxy_set_header Host $host;
    proxy_read_timeout 30s;
    # imageproxy already sets immutable Cache-Control; let it pass through
    proxy_pass_header Cache-Control;
}
```

### 5c. Linux Caddy
```caddy
handle_path /img/* {
    reverse_proxy 127.0.0.1:8080   # handle_path strips the /img prefix
}
```

Smoke-test through the public edge once wired:
```bash
b64=$(printf 'https://luatsumienbac.vn/media/<some-file>.jpg' | base64 | tr '+/' '-_' | tr -d '=')
curl -sI "https://luatsumienbac.vn/img/800x,avif,q55/$b64"   # 200, Content-Type: image/avif
curl -sI "https://luatsumienbac.vn/img/800x,webp/$b64"       # 200, Content-Type: image/webp
```

---

## 6. Caching

- **imageproxy disk cache** (`-cache <path>`): stores each transformed variant once,
  keyed on the full request URL (options + source). **Set this** — without it every hit
  re-fetches and re-encodes. Point it at fast local disk with room to grow
  (thousands of small AVIF/WebP files). Other backends: `memory:<MB>`, `s3://…`,
  `gcs://…`, `azure://…`, `redis://…`; multiple `-cache` values create a tiered cache.
- **Cloudflare**: responses carry a long immutable `Cache-Control`, so the edge caches
  them. Because each format has a **distinct URL** (from the `<picture>` element), edge
  caching is correct with no `Vary` gymnastics.

Cache invalidation: variant URLs are derived from the source filename + options. If an
editor **replaces** an image under the same filename, purge that path at Cloudflare (or
clear the imageproxy cache dir). New filenames need no purge.

---

## 7. Signed URLs (optional, defense-in-depth)

`-allowHosts` already prevents open-proxy abuse, so signing is optional. To enable it:

1. Generate a key: `openssl rand -hex 32`
2. **Server:** set `IMAGEPROXY_SIGNATUREKEY=<key>` (or `-signatureKey <key>`, or
   `-signatureKey @/etc/imageproxy.key`).
3. **Astro build:** set **`IMAGEPROXY_SIGNATURE_KEY=<same key>`** — note the build-side
   var name has an underscore (`SIGNATURE_KEY`) while the server-side one does not
   (`SIGNATUREKEY`); the **value must match**. `src/utils/imageResizer.ts` then appends
   an `s<sig>` option to every URL; without the build var it emits unsigned URLs (fine
   under `-allowHosts`).

---

## 8. Updating the fork

```bash
git pull                        # get new commits
go build -ldflags "-s -w" -o imageproxy ./cmd/imageproxy   # rebuild (Windows: build.ps1)
# restart:  Windows: imageproxy.exe -service restart    Linux: sudo systemctl restart imageproxy
```
To rebase the fork on a newer upstream, re-apply the small delta in
[FORK_NOTES.md](FORK_NOTES.md).

---

## 9. Troubleshooting

| Symptom | Cause & fix |
|---------|-------------|
| `403 requested URL is not allowed` | Source host not in `-allowHosts`, or a signature was expected but missing/wrong. Confirm `-allowHosts luatsumienbac.vn` and that the URL's origin matches. |
| AVIF returns `application/octet-stream` / 403 | You're on an **unpatched** imageproxy. This fork fixes it by setting an explicit `Content-Type` (Go's sniffer has no AVIF signature). Rebuild from this fork. |
| Blank/again `Cannot GET` at `/img/...` | Reverse proxy isn't stripping `/img`. IIS `{R:1}` / nginx trailing-slash `proxy_pass` / Caddy `handle_path` all strip it — verify the rule. |
| Slow first hit, fast after | Expected: the first request for a variant encodes then caches. Ensure `-cache` is set so it's paid once. AVIF is the heaviest; `q55`/speed keep it reasonable on ~1600 px originals. |
| 504 / cut-off on huge images | 30 s write timeout hit by a very large AVIF encode. Keep originals ≤ ~1600 px (the CMS already downsizes uploads). |
| Won't start: `listen failed` | Port already in use. Change `-addr` or free `:8080`. |
| `-service install` fails / access denied | Must run in an **elevated** (Administrator) PowerShell. |
| `install-service.ps1` aborts with `Can't open service!` / `NativeCommandError` | Old copy of the script on PowerShell 7.4+ (a harmless first-run "stop non-existent service" was treated as fatal). Pull the current script (it sets `$PSNativeCommandUseErrorActionPreference = $false`). |
| Service path has spaces / `&` (e.g. `…\Youth & Partners\…`) | Handled, but a space-free path like `C:\svc\imageproxy` is safest. The scripts pass args as an array so quoting is correct either way. |
| Service installed but keeps stopping | Check the `-logFile` and the Windows Event Log. Common causes: cache dir parent missing or not writable by LocalSystem, or `listen failed` (port in use). |

Enable `-verbose` during bring-up to log every fetch/transform and the served-from-cache
flag. When running as a service, combine it with `-logFile` to capture that output.
