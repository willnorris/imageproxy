# Fork notes — trungnsbkvn/imageproxy

This fork of [willnorris/imageproxy](https://github.com/willnorris/imageproxy) adds
**WebP and AVIF output encoding**, which upstream does not have (upstream decodes
WebP but only *encodes* JPEG/PNG/GIF/TIFF/BMP — see upstream issue #114).

It powers the on-the-fly `/img` resizer for **luatsumienbac.vn** (self-hosted media
served from `D:\media`, out of git). See the site repo's
`docs/RESIZER_PROVIDERS.md` and `docs/MEDIA_RESIZER.md`.

## Why fork instead of use upstream
We need modern-format output (WebP/AVIF ≈ 25–50 % smaller than JPEG) from a **single
pure-Go binary** that builds natively on Windows (the IIS host has no Docker and is
resource-tight). Upstream is the right base — pure Go, single binary, mature caching /
signing / host-allowlist — it just can't emit WebP/AVIF. This fork closes that one gap
without touching the fetch/cache/sign/allowlist machinery.

### Why not the maintainer's own WebP PR (#393)?
willnorris opened [PR #393](https://github.com/willnorris/imageproxy/pull/393) (May
2024) adding WebP output — but it's **still open/unmerged**, is **WebP-only (no AVIF)**,
and encodes via **`go-libwebp`, which needs cgo + the C libwebp library**. That
reintroduces exactly what we forked to avoid: a C toolchain to build, libwebp present at
runtime, and hard cross-compilation — i.e. no "one `.exe`, no dependencies" on the
Windows box. The PR also has acknowledged quality/size bugs (inverted `q`, "lossless"
~4× larger) that kept it from merging.

Our approach instead uses `gen2brain/webp` + `gen2brain/avif`, which compile libwebp and
libaom **to WASM** (run by wazero) — so it stays `CGO_ENABLED=0`, ships one static
binary, adds **AVIF** too, and is built + runtime-verified. Trade-off: WASM encode is
slower than native cgo libwebp, mitigated by the disk cache (encode once) and
pre-downsized originals. For a Docker-less, resource-tight Windows host that also wants
AVIF, pure-Go wins; native libwebp/libvips (or imgproxy) would only pull ahead on a big
Linux box optimizing purely for encode throughput.

## The delta (all pure-Go, `CGO_ENABLED=0`)

| File | Change |
|------|--------|
| `go.mod` | + `github.com/gen2brain/webp`, `github.com/gen2brain/avif` (libwebp/libaom via WASM/wazero — **no cgo**); + `github.com/kardianos/service` (pure-Go OS-service support) |
| `data.go` | `webp`/`avif` registered as parseable `format` options (`optFormatWEBP`, `optFormatAVIF`) |
| `transform.go` | `webp` + `avif` cases in the encode switch; `contentTypeForFormat()` helper; default qualities (WebP 80, AVIF 55, AVIF speed 8) |
| `imageproxy.go` | replay path sets an explicit `Content-Type` for known output formats — **required for AVIF**, because Go's `http.DetectContentType` has no AVIF signature and would mislabel it `application/octet-stream` (→ 403 under the default `image/*` content-type filter) |
| `cmd/imageproxy/service.go` *(new)* | native OS-service support: `-service install\|uninstall\|start\|stop\|restart` (Windows SCM / systemd / launchd — **no wrapper needed**; nssm still works as an external wrapper too) + `-logFile`. Foreground behaviour unchanged. |
| `cmd/imageproxy/main.go` | tail refactor: build the `http.Server`, then `runWithService(server)` instead of `server.Serve` directly (so it runs under a service manager or foreground). |

Nothing else changes: URL scheme, HMAC signing (`s` option), `allowHosts`, caching,
metrics, and all existing formats behave exactly as upstream.

## Deploying
Full Windows + Linux deployment guide (build, service via NSSM/systemd, IIS/nginx/Caddy
reverse proxy, caching, signing, troubleshooting): **[DEPLOY.md](DEPLOY.md)**.

## Build (single binary, any OS)
```bash
go mod tidy
CGO_ENABLED=0 go build -ldflags "-s -w" -o imageproxy .    # Linux
# Windows:
#   $env:CGO_ENABLED=0; go build -ldflags "-s -w" -o imageproxy.exe ./cmd/imageproxy
```
The binary is ~48 MB (embeds the libwebp + libaom WASM blobs). No runtime deps.

## Verified
Built with `CGO_ENABLED=0` and smoke-tested end-to-end (codercat.jpg @300px):

| option | Content-Type | bytes |
|--------|--------------|-------|
| `300,fit,avif` | `image/avif` | 3 860 |
| `300,fit,webp` | `image/webp` | 5 634 |
| `300,fit` (default) | `image/jpeg` | 18 362 |
| *(original)* | — | 28 863 |

## Usage (new options)
```
/{width},fit,webp/{base64url-or-plain remote URL}      → WebP
/{width},fit,avif,q55/{...}                            → AVIF at quality 55
```
Distinct URL per format → CDN-cache-correct; the site emits a `<picture>` so the
browser picks AVIF → WebP → JPEG. (imageproxy does **not** negotiate format from the
`Accept` header — that's intentional; its result cache is keyed on the URL only.)

## Keeping in sync with upstream
The delta is small and localized. To rebase on a newer upstream: re-apply the four
files above (the encode switch, the two option constants + parse case, the
`contentTypeForFormat` header fix, and the two go.mod requires). Watch for changes to
`transform.go`'s format switch and `imageproxy.go`'s response-replay block.
