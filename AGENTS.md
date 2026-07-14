# AGENTS.md — workbuddy-cli-proxy

## Purpose

Clean-room Go rewrite of the **workbuddy** [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) (CPA) plugin: wraps Tencent **CodeBuddy** (`copilot.tencent.com`) as an OpenAI-compatible provider for CPA. Original workbuddy design credited to Sliverkiss (`cpa-plugin`).

Module: `github.com/WslzGmzs/workbuddy-cli-proxy` · Plugin ID: **`workbuddy`** · Go **1.26+** · depends on `CLIProxyAPI/v7` (plugin ABI/API only).

## Layout

| Path | Role |
|------|------|
| `main.go` | Entire plugin: C ABI, auth (oauth + api_key), models, execute/stream, rewrites |
| `go.mod` / `go.sum` | Module + CPA SDK pin |
| `Makefile` | Local `build` / `package` (store-compatible zip) |
| `.github/workflows/build.yml` | Multi-arch CGO build + GitHub Release |
| `.github/scripts/package-release.go` | Zip library at root + sha256 line |
| `examples/workbuddy-api-key.json` | Manual API-key credential template |
| `registry.json` | CPA plugin-store registry (schema_version 1, github-release) |
| `docs/plugin-store-entry.json` | Same plugin object for official store PR |
| `README.md` | Install, credentials, store publishing |

Build artifacts (`*.so` / `*.dylib` / `*.dll` / `*.h` / `dist/` / `workbuddy_*.zip`) and credential files are gitignored — never commit tokens or API keys.

## Build & verify

Requires **CGO** + C toolchain; GOOS/GOARCH must match the CPA host.

```bash
make build                          # dist/workbuddy.<ext>
make package VERSION=0.2.0 GOOS=linux GOARCH=amd64
go vet ./...
```

Release: push tag `v0.2.0` → Actions publishes store-compatible zips + `checksums.txt`.

Keep CPA pin aligned with host (**v7.2.x**). Smoke: load plugin, `plugin_id=workbuddy`, `GET /v1/models`.

## Architecture (edit map)

Provider id: `workbuddy`. C exports: `cliproxy_plugin_init`, `cliproxyPluginCall`, `cliproxyPluginFree`, `cliproxyPluginShutdown`.

`handleMethod` → pluginabi methods:

- **Register / models**: `wbRegistration` (version from `pluginVersion` ldflag), `wbModels`
- **Auth**:
  - OAuth: `handleStartLogin` / `handlePollLogin` / `handleRefreshAuth` (state + cookie jar)
  - **Panel paste API key**: during login, CPA writes `.oauth-workbuddy-<state>.oauth`; poll reads it via `tryConsumePastedCredential` / `classifyPastedCredential` (URL vs raw key)
  - **API key file**: `parseStored` accepts `auth_type=api_key` (+ `api_key` / `apiKey`); refresh is no-op; `backendHeaders` sets Bearer + `X-API-Key`
- **Execute**: force upstream stream, then `aggregateCompletion` (code **11101**)
- **Execute stream**: async `host.stream.emit` / `close` when `stream_id` present

Upstream: `https://copilot.tencent.com` · chat `/v2/chat/completions`.

### Auth file shapes

OAuth (legacy login result):

```json
{"type":"workbuddy","auth":{"accessToken":"...","refreshToken":"...","expiresAt":0,"domain":"..."},"account":{"uid":"...","enterpriseId":"...","nickname":"..."}}
```

API key:

```json
{"type":"workbuddy","auth_type":"api_key","api_key":"...","user_id":"anonymous","domain":"copilot.tencent.com"}
```

## Gotchas (do not “simplify” away)

1. **Claude Code blocklist** — `sanitizeBlockedTemplates` (`CLI`→`CLI tool`, `Main branch`→`Default branch`).
2. **hy3 thinking** — prefix `hy3` → force `reasoning_effort=high`.
3. **Streaming** — true streaming via host emit; emit failure stops pump.
4. **SSE framing** — `clientNeedsSSEFrame` for non-`/v1/chat/completions` paths.
5. **Login cookies** — one jar per login `state` (`loginCtx`).
6. **Chunk cleanup** — `cleanChunkJSON` drops empty delta fields.
7. **Credentials** — never log/commit `workbuddy.json` or API keys.
8. **Store packaging** — zip must contain **only** `workbuddy.<ext>` at root; asset name `workbuddy_<ver>_<goos>_<goarch>.zip`.
9. **API key vs OAuth** — do not send refresh headers for api_key mode; do not require `accessToken` when `api_key` is set.

## Conventions

- Prefer helpers in `main.go` unless size forces a split.
- Inject version with `-X main.pluginVersion=...` on release builds.
- Keep model ids stable; context lengths in `wbModels`.
- Plugin store `repository` field must be exact `https://github.com/WslzGmzs/workbuddy-cli-proxy` (no trailing slash, no `.git`).
- Root `registry.json` is the private `store-sources` entry point; version field is display fallback only — real version comes from GitHub latest release tag `v*`.

## Docs to read first

- `README.md` — install, credentials, store PR flow
- `docs/plugin-store-entry.json` — registry entry
- Comments on `rewriteSystemForUpstream`, `forceMaxThinking`, `handleExecStream`, `clientNeedsSSEFrame`, `parseStored` / `backendHeaders`
